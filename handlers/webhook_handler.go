// handlers/webhook_handler.go
package handlers

import (
	"gitlab-mr-review/entities"
	"gitlab-mr-review/services"

	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct {
	reviewService *services.ReviewService
	gitlabToken  string
}

func NewWebhookHandler(reviewService *services.ReviewService, gitlabToken string) *WebhookHandler {
	return &WebhookHandler{
		reviewService: reviewService,
		gitlabToken:  gitlabToken,
	}
}

func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	var mrData entities.MergeRequestData
	if err := c.BodyParser(&mrData); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if mrData.ObjectKind != "merge_request" {
		return c.Status(200).JSON(fiber.Map{"status": "ignored"})
	}

	review, err := h.reviewService.AnalyzeMergeRequest(&mrData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"review": review,
	})
}