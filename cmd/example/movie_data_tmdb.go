package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	// Твой Bearer токен TMDB
	bearerToken := "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiIxZDc0M2Q5N2E4OTQyYzZmMGI5Y2UyZWQxMzM5YzVlZiIsIm5iZiI6MTc1NjU4MjMyMy4wNCwic3ViIjoiNjhiMzUxYjMzNzg4OWI4YzU3MWYxZmRmIiwic2NvcGVzIjpbImFwaV9yZWFkIl0sInZlcnNpb24iOjF9.oRoW11hk_Us9oQ0Y5au7O9EgkybhEPfN4XEd-T5rcE4" // <-- замени на свой токен

	// Начальный movie ID
	startID := 30000
	numRequests := 10000

	for i := 0; i < numRequests; i++ {
		movieID := startID + i
		//url := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d", movieID)
		//https: //api.themoviedb.org/3/movie/1?append_to_response=keywords,credits&language=en-US
		//https: //api.themoviedb.org/3/movie/2?append_to_response=keywords,credits&language=en-US
		//https: //api.themoviedb.org/3/movie/2?append_to_response=keywords%2Ccredits&language=en-US
		url := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d?append_to_response=keywords,credits&language=en-US", movieID)
		//fmt.Println(url)
		/**
		curl --request GET \
		     --url 'https://api.themoviedb.org/3/movie/11?append_to_response=keywords%2Ccredits&language=en-US' \
		     --header 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiIxZDc0M2Q5N2E4OTQyYzZmMGI5Y2UyZWQxMzM5YzVlZiIsIm5iZiI6MTc1NjU4MjMyMy4wNCwic3ViIjoiNjhiMzUxYjMzNzg4OWI4YzU3MWYxZmRmIiwic2NvcGVzIjpbImFwaV9yZWFkIl0sInZlcnNpb24iOjF9.oRoW11hk_Us9oQ0Y5au7O9EgkybhEPfN4XEd-T5rcE4' \
		     --header 'accept: application/json'
		*/
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request for movie %d: %v\n", movieID, err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+bearerToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error fetching movie %d: %v\n", movieID, err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("Error reading response for movie %d: %v\n", movieID, err)
			continue
		}

		if resp.StatusCode != 200 {
			fmt.Printf("Non-200 response for movie %d: %s, body: %s\n", movieID, resp.Status, body)
			continue
		}
		//  /home/torys/neoworkspace/2_projects/in_progress/vectrain/movie_data2
		// Записываем JSON в файл
		filename := fmt.Sprintf("/home/torys/neoworkspace/2_projects/in_progress/vectrain/movie_data2/movie_%d.json", movieID)
		err = os.WriteFile(filename, body, 0644)
		if err != nil {
			fmt.Printf("Error writing file for movie %d: %v\n", movieID, err)
			continue
		}

		fmt.Printf("Saved movie %d to %s\n", movieID, filename)
		time.Sleep(200 * time.Millisecond)
	}
}
