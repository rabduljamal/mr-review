// services/review_service.go
package services

import (
	"fmt"
	"strings"
	"time"

	"gitlab-mr-review/entities"
	"gitlab-mr-review/repositories"
)

type ReviewService struct {
	weaviateRepo *repositories.WeaviateRepository
	groqService  *GroqService
}

func NewReviewService(weaviateRepo *repositories.WeaviateRepository, groqService *GroqService) *ReviewService {
	return &ReviewService{
		weaviateRepo: weaviateRepo,
		groqService:  groqService,
	}
}

func (s *ReviewService) AnalyzeMergeRequest(mr *entities.MergeRequestData) (string, error) {
	// Format code changes
	var changes strings.Builder
	for file, change := range mr.ObjectAttributes.Changes {
		changes.WriteString(fmt.Sprintf("File: %s\n", file))
		changes.WriteString(fmt.Sprintf("Diff:\n%s\n\n", change.Diff))
	}

	codeDiff := changes.String()

	// Get similar review
	similarReview, err := s.weaviateRepo.FindSimilarReview(codeDiff)
	if err != nil {
		similarReview = "No similar PR found."
	}

	// Create prompt
	prompt := fmt.Sprintf(`
You are a senior software engineer reviewing a pull request.

Here is the code diff:
%s

Based on past PR reviews, here is a relevant comment:
%s

Now generate a review including:
1. Code quality assessment
2. Best practices evaluation
3. Security concerns
4. Performance implications
5. Specific suggestions for improvement
6. Any potential edge cases or risks

Please provide a structured, detailed review.
`, codeDiff, similarReview)

	// Generate review
	review, err := s.groqService.GenerateReview(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate review: %v", err)
	}

	// Store review
	reviewModel := &entities.Review{
		Project:     mr.Project.Name,
		Title:       mr.ObjectAttributes.Title,
		Description: mr.ObjectAttributes.Description,
		Changes:     codeDiff,
		Review:      review,
		Timestamp:   time.Now(),
	}

	if err := s.weaviateRepo.StoreReview(reviewModel); err != nil {
		// Log error but continue
		fmt.Printf("Failed to store review: %v\n", err)
	}

	return review, nil
}