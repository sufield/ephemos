package main

import (
	"fmt"
	"log"

	"github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
	log.Println("🧪 Testing Enhanced Error Handling with errorx")
	log.Println("================================================")

	// Test validation error
	log.Println("\n1. Validation Error Test:")
	err1 := ephemos.NewEnhancedValidationError("user.email", "invalid@", "email domain missing")
	fmt.Printf("   Error: %v\n", err1)
	fmt.Printf("   Is validation error: %t\n", ephemos.IsEnhancedValidationError(err1))
	fmt.Printf("   Field: %s\n", ephemos.GetEnhancedErrorField(err1))
	fmt.Printf("   Value: %v\n", ephemos.GetEnhancedErrorValue(err1))
	fmt.Printf("   Code: %s\n", ephemos.GetEnhancedErrorCode(err1))

	// Test system error with timeout
	log.Println("\n2. Timeout Error Test:")
	err2 := ephemos.NewTimeoutError("database_query", "query timed out after 30s")
	fmt.Printf("   Error: %v\n", err2)
	fmt.Printf("   Is timeout: %t\n", ephemos.IsTimeoutError(err2))
	fmt.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(err2))
	fmt.Printf("   Operation: %s\n", ephemos.GetEnhancedErrorOperation(err2))
	fmt.Printf("   Code: %s\n", ephemos.GetEnhancedErrorCode(err2))

	// Test temporary error
	log.Println("\n3. Temporary Error Test:")
	err3 := ephemos.NewTemporaryError("api-gateway", "service temporarily unavailable")
	fmt.Printf("   Error: %v\n", err3)
	fmt.Printf("   Is temporary: %t\n", ephemos.IsTemporaryError(err3))
	fmt.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(err3))
	fmt.Printf("   Service: %s\n", ephemos.GetEnhancedErrorService(err3))

	// Test predefined errors
	log.Println("\n4. Predefined Error Test:")
	err4 := ephemos.ErrEnhancedConnectionFailed
	fmt.Printf("   Error: %v\n", err4)
	fmt.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(err4))
	fmt.Printf("   Code: %s\n", ephemos.GetEnhancedErrorCode(err4))

	// Test error wrapping
	log.Println("\n5. Error Wrapping Test:")
	originalErr := fmt.Errorf("network connection refused")
	wrappedErr := ephemos.WrapWithEnhancedContext(originalErr, ephemos.ConnectionError, "failed to connect to database")
	fmt.Printf("   Wrapped Error: %v\n", wrappedErr)
	fmt.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(wrappedErr))

	// Test error decoration
	log.Println("\n6. Error Decoration Test:")
	baseErr := ephemos.EnhancedValidationError.New("base validation error")
	decoratedErr := ephemos.DecorateError(baseErr, "additional context")
	fmt.Printf("   Decorated Error: %v\n", decoratedErr)
	fmt.Printf("   Is validation error: %t\n", ephemos.IsEnhancedValidationError(decoratedErr))

	log.Println("\n✅ All enhanced error tests completed successfully!")
	log.Println("Enhanced error handling with errorx is working correctly")
}

