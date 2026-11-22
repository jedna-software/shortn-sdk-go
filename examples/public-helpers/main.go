// Command public-helpers uses the endpoints that need no API key: URL
// validation, a QR code and the plan catalog.
//
//	go run ./examples/public-helpers
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	shortn "github.com/jedna-software/shortn-sdk-go"
)

func main() {
	client, err := shortn.NewPublicClient()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	validation, err := client.Public.ValidateURL(ctx, "example.com/pricing")
	if err != nil {
		log.Fatal(err)
	}
	if !validation.Valid {
		log.Fatalf("invalid URL: %s (%s)", validation.Error, validation.Code)
	}
	fmt.Println("normalized:", validation.Normalized)

	png, err := client.Public.QRCode(ctx, shortn.QRCodeParams{URL: validation.Normalized, Format: "png"})
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("qr.png", png, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote qr.png (%d bytes)\n", len(png))

	catalog, err := client.Public.Plans(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for id, plan := range catalog.Plans {
		fmt.Printf("%s: %s, %d %s per %s\n", id, plan.Name, plan.Price, plan.Currency, plan.Interval)
	}
}
