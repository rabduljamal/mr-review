package main

import (
	"log"
	"os"

	"gitlab-mr-review/entities"
	"gitlab-mr-review/handlers"
	"gitlab-mr-review/repositories"
	"gitlab-mr-review/services"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func startServer(config entities.Config) {

	// Initialize repositories
	weaviateRepo, err := repositories.NewWeaviateRepository(config.WeaviateURL, config.WeaviateAPIKey)
	if err != nil {
		log.Fatal(err)
	}

	// Create schema
	if err := weaviateRepo.CreateSchema(); err != nil {
		log.Fatal(err)
	}

	// Initialize services
	groqService := services.NewGroqService(config.GroqAPIKey)
	reviewService := services.NewReviewService(weaviateRepo, groqService)

	// Initialize handlers
	webhookHandler := handlers.NewWebhookHandler(reviewService, config.GitlabToken)

	// Setup Fiber app
	app := fiber.New()

	// Routes
	app.Post("/webhook/gitlab", webhookHandler.HandleWebhook)

	// Start server
		log.Fatal(app.Listen(":" + config.Port))
}

func main() {
	err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
	config := entities.Config{
		GitlabToken:    os.Getenv("GITLAB_SECRET_TOKEN"),
		GroqAPIKey:     os.Getenv("GROQ_API_KEY"),
		WeaviateURL:    os.Getenv("WEAVIATE_URL"),
		WeaviateAPIKey: os.Getenv("WEAVIATE_API_KEY"),
		Port:           os.Getenv("PORT"),
	}

	if config.Port == "" {
		config.Port = "3000"
	}

	startServer(config)
}