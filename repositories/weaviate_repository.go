// repositories/weaviate_repository.go
package repositories

import (
	"context"
	"strings"
	"time"

	"gitlab-mr-review/entities"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

type WeaviateRepository struct {
	client *weaviate.Client
}

func NewWeaviateRepository(url, apiKey string) (*WeaviateRepository, error) {
	cfg := weaviate.Config{
		Host:       url,
		Scheme:     "http",
		AuthConfig: auth.ApiKey{Value: apiKey},
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &WeaviateRepository{client: client}, nil
}

func (r *WeaviateRepository) CreateSchema() error {
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

	err := r.client.Schema().ClassCreator().WithClass(classObj).Do(ctx)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	return nil
}

func (r *WeaviateRepository) StoreReview(review *entities.Review) error {
	ctx := context.Background()

	properties := map[string]interface{}{
		"project":     review.Project,
		"title":       review.Title,
		"description": review.Description,
		"changes":     review.Changes,
		"review":      review.Review,
		"timestamp":   review.Timestamp.Format(time.RFC3339),
	}

	_, err := r.client.Data().Creator().
		WithClassName("MergeRequestReview").
		WithProperties(properties).
		Do(ctx)

	return err
}

func (r *WeaviateRepository) FindSimilarReview(codeDiff string) (string, error) {
	ctx := context.Background()

	whereFilter := filters.Where().
		WithOperator("And").
		WithOperands([]*filters.WhereBuilder{
			filters.Where().
				WithPath([]string{"changes"}).
				WithOperator("Like").
				WithValueString(codeDiff),
		})


	result, err := r.client.GraphQL().Get().
		WithClassName("MergeRequestReview").
		WithFields(graphql.Field{Name: "review"}).
		WithWhere(whereFilter).
		WithLimit(1).
		Do(ctx)

	if err != nil {
		return "", err
	}

	reviews, ok := result.Data["Get"].(map[string]interface{})["MergeRequestReview"].([]interface{})
	if !ok || len(reviews) == 0 {
		return "", nil
	}

	review := reviews[0].(map[string]interface{})["review"].(string)
	return review, nil
}