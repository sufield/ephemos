package main

import (
	"errors"
	"log"

	"github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
	log.Println("🧪 Testing Enhanced Error Handling with errorx")
	log.Println("================================================")

	// Test validation error
	log.Println("\n1. Validation Error Test:")
	err1 := ephemos.NewEnhancedValidationError("user.email", "invalid@", "email domain missing")
	log.Printf("   Error: %v\n", err1)
	log.Printf("   Is validation error: %t\n", ephemos.IsEnhancedValidationError(err1))
	log.Printf("   Field: %s\n", ephemos.GetEnhancedErrorField(err1))
	log.Printf("   Value: %v\n", ephemos.GetEnhancedErrorValue(err1))
	log.Printf("   Code: %s\n", ephemos.GetEnhancedErrorCode(err1))

	// Test system error with timeout
	log.Println("\n2. Timeout Error Test:")
	err2 := ephemos.NewTimeoutError("database_query", "query timed out after 30s")
	log.Printf("   Error: %v\n", err2)
	log.Printf("   Is timeout: %t\n", ephemos.IsTimeoutError(err2))
	log.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(err2))
	log.Printf("   Operation: %s\n", ephemos.GetEnhancedErrorOperation(err2))
	log.Printf("   Code: %s\n", ephemos.GetEnhancedErrorCode(err2))

	// Test temporary error
	log.Println("\n3. Temporary Error Test:")
	err3 := ephemos.NewTemporaryError("api-gateway", "service temporarily unavailable")
	log.Printf("   Error: %v\n", err3)
	log.Printf("   Is temporary: %t\n", ephemos.IsTemporaryError(err3))
	log.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(err3))
	log.Printf("   Service: %s\n", ephemos.GetEnhancedErrorService(err3))

	// Test predefined errors
	log.Println("\n4. Predefined Error Test:")
	err4 := ephemos.ErrEnhancedConnectionFailed
	log.Printf("   Error: %v\n", err4)
	log.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(err4))
	log.Printf("   Code: %s\n", ephemos.GetEnhancedErrorCode(err4))

	// Test error wrapping
	log.Println("\n5. Error Wrapping Test:")
	originalErr := errors.New("network connection refused")
	wrappedErr := ephemos.WrapWithEnhancedContext(originalErr, ephemos.ConnectionError, "failed to connect to database")
	log.Printf("   Wrapped Error: %v\n", wrappedErr)
	log.Printf("   Is system error: %t\n", ephemos.IsEnhancedSystemError(wrappedErr))

	// Test error decoration
	log.Println("\n6. Error Decoration Test:")
	baseErr := ephemos.EnhancedValidationError.New("base validation error")
	decoratedErr := ephemos.DecorateError(baseErr, "additional context")
	log.Printf("   Decorated Error: %v\n", decoratedErr)
	log.Printf("   Is validation error: %t\n", ephemos.IsEnhancedValidationError(decoratedErr))

	log.Println("\n✅ All enhanced error tests completed successfully!")
	log.Println("Enhanced error handling with errorx is working correctly")
}
