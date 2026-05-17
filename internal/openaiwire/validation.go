package openaiwire

import "fmt"

func ValidateChatCompletionsRequest(req ChatCompletionsRequest) error {
	for i, message := range req.Messages {
		if !message.Content.IsParts() {
			continue
		}

		for j, part := range message.Content.Parts() {
			if part.Type == "" {
				return fmt.Errorf("messages[%d].content[%d].type is required", i, j)
			}
			switch part.Type {
			case "text":
			case "image_url":
				if part.ImageURL == nil || part.ImageURL.URL == "" {
					return fmt.Errorf("messages[%d].content[%d].image_url.url is required", i, j)
				}
			default:
				return fmt.Errorf("messages[%d].content[%d].type %q is not supported", i, j, part.Type)
			}
		}
	}

	return nil
}
