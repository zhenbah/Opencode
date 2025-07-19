package integration

import (
	"fmt"
	"net/http"
	"testing"
)

func TestRootHandler(t *testing.T) {
	client := &http.Client{}

	resp, err := client.Get("http://localhost:3000")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	fmt.Printf("response content is %s\n", resp.Body)

}
