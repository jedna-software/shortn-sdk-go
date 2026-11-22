// Command create-conversion-link creates a conversion-tracked link in a
// workspace and validates a test lead for one of its clicks.
//
//	SHORTN_API_KEY=sk_live_... SHORTN_WORKSPACE=<workspace public_id> \
//	SHORTN_CLICK_ID=clk_... go run ./examples/create-conversion-link
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	shortn "github.com/jedna-software/shortn-sdk-go"
)

func main() {
	client, err := shortn.NewClient(os.Getenv("SHORTN_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	link, err := client.Links.Create(ctx, shortn.CreateLinkParams{
		URL:               "https://example.com/pricing",
		Title:             "Pricing",
		WorkspacePublicID: os.Getenv("SHORTN_WORKSPACE"),
		ConversionEnabled: shortn.Bool(true),
		UTMSource:         "newsletter",
		UTMMedium:         "email",
		UTMCampaign:       "spring_launch",
		Attribution: &shortn.LinkAttribution{
			Type:       "campaign",
			ExternalID: "spring-launch",
			Label:      "Spring launch",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("short URL:", link.ShortURL)

	if clickID := os.Getenv("SHORTN_CLICK_ID"); clickID != "" {
		check, err := client.Conversions.ValidateLead(ctx, shortn.TrackLeadParams{
			ClickID:        clickID,
			EventName:      "Sign up",
			LeadCategory:   "signup",
			IdempotencyKey: "signup_example",
			TestMode:       true,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("lead would be recorded:", check.Result.WouldTrack)
	}
}
