package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

type Config struct {
	GitlabToken    string
	GroqAPIKey     string
	WeaviateURL    string
	WeaviateAPIKey string
}

type App struct {
	config         Config
	weaviateClient *weaviate.Client
}

type MergeRequestData struct {
	ObjectKind string `json:"object_kind"`
	Project    struct {
		Name string `json:"name"`
	} `json:"project"`
	ObjectAttributes struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Changes     map[string]struct {
			Diff    string `json:"diff"`
			NewPath string `json:"new_path"`
			OldPath string `json:"old_path"`
		} `json:"changes"`
	} `json:"object_attributes"`
}

type Review struct {
	Project     string    `json:"project"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Changes     string    `json:"changes"`
	Review      string    `json:"review"`
	Timestamp   time.Time `json:"timestamp"`
}

func main2() {
	err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
	
	config := Config{
		GitlabToken:    os.Getenv("GITLAB_SECRET_TOKEN"),
		GroqAPIKey:     os.Getenv("GROQ_API_KEY"),
		WeaviateURL:    os.Getenv("WEAVIATE_URL"),
		WeaviateAPIKey: os.Getenv("WEAVIATE_API_KEY"),
	}

	log.Print(config.WeaviateURL)
	app := NewApp(config)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func NewApp(config Config) *App {
	cfg := weaviate.Config{
		Host:       config.WeaviateURL,
		Scheme:     "http",
		AuthConfig: auth.ApiKey{Value: config.WeaviateAPIKey},
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &App{
		config:         config,
		weaviateClient: client,
	}
}

func (a *App) Run() error {
	app := fiber.New()

	// Create Weaviate schema on startup
	if err := a.createWeaviateSchema(); err != nil {
		return err
	}

	app.Post("/webhook/gitlab", a.verifyGitlabWebhook, a.handleWebhook)
	app.Post("/similar-reviews", a.findSimilarReviews)

	return app.Listen(":3000")
}

func (a *App) verifyGitlabWebhook(c *fiber.Ctx) error {
	if a.config.GitlabToken == "" {
		return c.Next()
	}

	signature := c.Get("X-Gitlab-Token")
	if signature == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Missing signature"})
	}

	mac := hmac.New(sha256.New, []byte(a.config.GitlabToken))
	mac.Write(c.Body())
	// expectedMAC := hex.EncodeToString(mac.Sum(nil))

	// if !hmac.Equal([]byte(signature), []byte(fmt.Sprintf("sha256=%s", expectedMAC))) {
	// 	return c.Status(401).JSON(fiber.Map{"error": "Invalid signature"})
	// }

	return c.Next()
}

func (a *App) createWeaviateSchema() error {
	ctx := context.Background()

	classObj := &models.Class{
		Class: "MergeRequestReview",
		Properties: []*models.Property{
			{Name: "project", DataType: []string{"string"}},
			{Name: "title", DataType: []string{"text"}},
			{Name: "description", DataType: []string{"text"}},
			{Name: "changes", DataType: []string{"text"}},
			{Name: "review", DataType: []string{"text"}},
			{Name: "timestamp", DataType: []string{"date"}},
		},
	}

	err := a.weaviateClient.Schema().ClassCreator().WithClass(classObj).Do(ctx)
	if err != nil {
		// Ignore error if class already exists
		if !strings.Contains(err.Error(), "already exists") {
			return err
		}
	}
	return nil
}

func (a *App) analyzeMergeRequest(mr MergeRequestData) (string, error) {
	// Create the analysis request for Groq
	analysisRequest := struct {
		Project     string
		Title       string
		Description string
		Changes     map[string]struct {
			Diff    string `json:"diff"`
			NewPath string `json:"new_path"`
			OldPath string `json:"old_path"`
		}
	}{
		Project:     mr.Project.Name,
		Title:       mr.ObjectAttributes.Title,
		Description: mr.ObjectAttributes.Description,
		Changes:     mr.ObjectAttributes.Changes,
	}

	// TODO: Implement actual Groq API call
	// For now, returning a placeholder response
	return fmt.Sprintf("Analysis for MR: %s", analysisRequest.Title), nil
}

func (a *App) storeInWeaviate(review Review) error {
	ctx := context.Background()

	properties := map[string]interface{}{
		"project":     review.Project,
		"title":       review.Title,
		"description": review.Description,
		"changes":     review.Changes,
		"review":      review.Review,
		"timestamp":   review.Timestamp.Format(time.RFC3339),
	}

	_, err := a.weaviateClient.Data().Creator().
		WithClassName("MergeRequestReview").
		WithProperties(properties).
		Do(ctx)

	return err
}

func (a *App) handleWebhook(c *fiber.Ctx) error {
	var mrData MergeRequestData
	if err := c.BodyParser(&mrData); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if mrData.ObjectKind != "merge_request" {
		return c.Status(200).JSON(fiber.Map{"status": "ignored"})
	}

	reviewContent, err := a.analyzeMergeRequest(mrData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	log.Println(reviewContent)
	review := Review{
		Project:     mrData.Project.Name,
		Title:       mrData.ObjectAttributes.Title,
		Description: mrData.ObjectAttributes.Description,
		Changes:     fmt.Sprintf("%v", mrData.ObjectAttributes.Changes),
		Review:      reviewContent,
		Timestamp:   time.Now(),
	}

	if err := a.storeInWeaviate(review); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"review": reviewContent,
	})
}

func (a *App) findSimilarReviews(c *fiber.Ctx) error {
	ctx := context.Background()
	
	var req struct {
		Query string `json:"query"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	whereFilter := filters.Where().
		WithOperator("And").
		WithOperands([]*filters.WhereBuilder{
			filters.Where().
				WithPath([]string{"title"}).
				WithOperator("Like").
				WithValueString(req.Query),
		})

	result, err := a.weaviateClient.GraphQL().Get().
		WithClassName("MergeRequestReview").
		WithFields(graphql.Field{Name: "project"}, graphql.Field{Name: "title"}, graphql.Field{Name: "review"}).
		WithWhere(whereFilter).
		WithLimit(5).
		Do(ctx)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}