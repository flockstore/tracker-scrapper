package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-rod/rod"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println(`{"error": "Please provide a tracking number as an argument"}`)
		return
	}
	trackingNumber := os.Args[1]

	// URL with the tracking number in the query string
	url := fmt.Sprintf("https://coordinadora.com/rastreo/rastreo-de-guia/detalle-de-rastreo-de-guia/?guia=%s", trackingNumber)

	// Connect to the browser
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// Create a new page
	page := browser.MustPage(url)

	// Setup request hijacking
	router := page.HijackRequests()
	defer router.MustStop()

	// Channel to signal when we've captured the response
	done := make(chan string)

	// Intercept the API call
	// The wildcard * matches the beginning of the URL path or domain
	// Matching pattern based on user request: /wp-json/rgc/v1/detail_tracking
	router.MustAdd("*/wp-json/rgc/v1/detail_tracking*", func(ctx *rod.Hijack) {
		// Continue the request and load the response
		// Pass http.DefaultClient to avoid panic
		if err := ctx.LoadResponse(http.DefaultClient, true); err != nil {
			return
		}

		// Get response body
		body := ctx.Response.Body()

		// Send to channel
		done <- body
	})

	// Start the router
	go router.Run()

	// Wait for the response to come through the channel
	// We might want to add a timeout, but for this task simple blocking is fine
	jsonOutput := <-done

	// Print the collected JSON
	fmt.Println(jsonOutput)
}
