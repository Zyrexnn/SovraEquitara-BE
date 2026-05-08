package main
import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)
func main() {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:3000/api/leaderboard")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Status: %s\nBody: %s\n", resp.Status, string(body))
}
