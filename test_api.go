package main
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)
func main() {
	resp, err := http.Get("http://localhost:3000/api/leaderboard")
	if err != nil { log.Fatal(err) }
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
}
