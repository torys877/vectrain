package main

import (
	"flag"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/torys877/vectrain/pkg/types"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"encoding/json"
)

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Cast struct {
	Adult              bool    `json:"adult"`
	Gender             int     `json:"gender"`
	ID                 int     `json:"id"`
	KnownForDepartment string  `json:"known_for_department"`
	Name               string  `json:"name"`
	OriginalName       string  `json:"original_name"`
	Popularity         float64 `json:"popularity"`
	ProfilePath        string  `json:"profile_path"`
	CastID             int     `json:"cast_id"`
	Character          string  `json:"character"`
	CreditID           string  `json:"credit_id"`
	Order              int     `json:"order"`
}
type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type KeywordsWrapper struct {
	Keywords []Keyword `json:"keywords"`
}

type ProductionCompany struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type ProductionCountry struct {
	ISO31661 string `json:"iso_3166_1"`
	Name     string `json:"name"`
}

type SpokenLanguage struct {
	EnglishName string `json:"english_name"`
	Iso6391     string `json:"iso_639_1"`
	Name        string `json:"name"`
}

type Movie struct {
	Adult               bool                `json:"adult"`
	BackdropPath        string              `json:"backdrop_path"`
	BelongsToCollection interface{}         `json:"belongs_to_collection"`
	Budget              int                 `json:"budget"`
	Genres              []Genre             `json:"genres"`
	Keywords            KeywordsWrapper     `json:"keywords"`
	Cast                []Cast              `json:"cast"`
	Homepage            string              `json:"homepage"`
	ID                  int                 `json:"id"`
	IMDBID              string              `json:"imdb_id"`
	OriginCountry       []string            `json:"origin_country"`
	OriginalLanguage    string              `json:"original_language"`
	OriginalTitle       string              `json:"original_title"`
	Overview            string              `json:"overview"`
	Popularity          float64             `json:"popularity"`
	PosterPath          string              `json:"poster_path"`
	ProductionCompanies []ProductionCompany `json:"production_companies"`
	ProductionCountries []ProductionCountry `json:"production_countries"`
	ReleaseDate         string              `json:"release_date"`
	Revenue             int                 `json:"revenue"`
	Runtime             int                 `json:"runtime"`
	SpokenLanguages     []SpokenLanguage    `json:"spoken_languages"`
	Status              string              `json:"status"`
	Tagline             string              `json:"tagline"`
	Title               string              `json:"title"`
	Video               bool                `json:"video"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
}

type MessageMovie struct {
	Title    string
	Overview string
}

func main() {
	// Флаг для пути к папке
	dirPtr := flag.String("dir", ".", "Path to folder containing movie JSON files")
	flag.Parse()

	startID := 1
	endID := 30000
	topic := "production1"

	var movies []Movie

	for id := startID; id <= endID; id++ {
		filename := filepath.Join(*dirPtr, fmt.Sprintf("movie_%d.json", id))
		data, err := os.ReadFile(filepath.Clean(filename))
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", filename, err)
			continue
		}

		var movie Movie
		err = json.Unmarshal(data, &movie)
		if err != nil {
			fmt.Printf("Error parsing JSON from %s: %v\n", filename, err)
			continue
		}

		movies = append(movies, movie)
		//fmt.Printf("Loaded movie: %s\n", movie.Title)
	}

	fmt.Printf("Total movies loaded: %d\n", len(movies))

	if len(movies) == 0 {
		os.Exit(1)
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "127.0.0.1:9092",
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	for _, movie := range movies {
		//fmt.Println(movie.Title)

		var genreNames []string
		for _, g := range movie.Genres {
			genreNames = append(genreNames, g.Name)
		}

		var keywords []string
		for _, k := range movie.Keywords.Keywords {
			keywords = append(keywords, k.Name)
		}

		message := fmt.Sprintf(
			"Title: %s\nOverview: %s\nGenres: %s\nTagline: %s\n Keywords: %s\n",
			movie.Title,
			movie.Overview,
			strings.Join(genreNames, ", "),
			movie.Tagline,
			strings.Join(keywords, ", "),
		)
		payload := map[string]string{
			"title":  movie.Title,
			"genres": strings.Join(genreNames, ", "),
			"year":   movie.ReleaseDate,
			"rating": strconv.FormatFloat(movie.VoteAverage, 'f', 2, 64),
		}

		entity := &types.Entity{
			UUID:    uuid.New().String(),
			Text:    message,
			Payload: payload,
		}

		// serializing in JSON for kafka
		valueBytes, err := json.Marshal(entity)
		if err != nil {
			log.Printf("Failed to marshal entity: %v", err)
			continue
		}

		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          valueBytes,
		}, nil)
		if err != nil {
			log.Printf("Failed to produce message: %s", err)
		}
	}

	// wait all messages to be sent
	producer.Flush(15 * 1000)
	fmt.Println("All messages sent")
}
