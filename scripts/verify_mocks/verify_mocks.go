package main

import (
	"fmt"
	"log"

	"github.com/kengibson1111/go-aiprovider/utils"
)

func main() {
	fmt.Println("Verifying Mock Infrastructure...")

	// Test basic mock functionality
	fmt.Print("Testing basic mock infrastructure... ")
	if err := utils.DemoMockInfrastructure(); err != nil {
		log.Fatalf("Basic mock infrastructure test failed: %v", err)
	}
	fmt.Println("✓ PASSED")

	// Test configurable error responses
	fmt.Print("Testing configurable error responses... ")
	if err := utils.DemoConfigurableErrorResponses(); err != nil {
		log.Fatalf("Configurable error responses test failed: %v", err)
	}
	fmt.Println("✓ PASSED")

	// Test special API responses
	fmt.Print("Testing special API responses... ")
	if err := utils.DemoSpecialAPIResponses(); err != nil {
		log.Fatalf("Special API responses test failed: %v", err)
	}
	fmt.Println("✓ PASSED")

	fmt.Println("\n🎉 All mock infrastructure tests passed!")
	fmt.Println("\nMock Infrastructure Features Verified:")
	fmt.Println("✓ MockHTTPClient for simulating HTTP responses")
	fmt.Println("✓ MockNetworkMonitor for testing network-dependent code")
	fmt.Println("✓ Test helpers for common mock scenarios")
	fmt.Println("✓ Configurable error responses in mocks")
	fmt.Println("✓ Request history and counting")
	fmt.Println("✓ Delay simulation")
	fmt.Println("✓ Context cancellation support")
	fmt.Println("✓ Default responses and errors")
	fmt.Println("✓ Network status callbacks")
	fmt.Println("✓ Endpoint connectivity testing")
	fmt.Println("✓ Special API response helpers (Claude, OpenAI)")
	fmt.Println("✓ Rate limiting and authentication error responses")
	fmt.Println("✓ Assertion helpers for test validation")
}
