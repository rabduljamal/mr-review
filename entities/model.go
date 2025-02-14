package entities

import "time"

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
	ID          string    `json:"id"`
	Project     string    `json:"project"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Changes     string    `json:"changes"`
	Review      string    `json:"review"`
	Timestamp   time.Time `json:"timestamp"`
}

type Config struct {
	GitlabToken    string
	GroqAPIKey     string
	WeaviateURL    string
	WeaviateAPIKey string
	Port           string
}
