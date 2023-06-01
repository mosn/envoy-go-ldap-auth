package test

import (
	"net/http"
	"testing"
)

func TestE2EBind(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:10000/", nil)

	resp1, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %v", resp1.StatusCode)
	}

	req.SetBasicAuth("unknown", "dogood")
	resp2, err := http.DefaultClient.Do(req)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %v", resp2.StatusCode)
	}

	req.SetBasicAuth("hackers", "unknown")
	resp3, err := http.DefaultClient.Do(req)
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %v", resp3.StatusCode)
	}

	req.SetBasicAuth("hackers", "dogood")
	resp4, err := http.DefaultClient.Do(req)
	defer resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %v", resp4.StatusCode)
	}

	t.Log("E2E test passed")

}
