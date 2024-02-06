package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"math/rand"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type Song struct {
	Song   string
	Artist string
	Writer string
	Album  string
	Year   string
	Rating int
}

type Songs map[string]Song

var songs Songs
var voteCounter int

func calculateExpectedScore(rating1 int, rating2 int) float64 {
	return 1 / (1 + math.Pow(10, float64((rating2-rating1)/400)))
}

// Calculate new Elo rating.
// 1.0 = rating1 wins, 0.5 = draw, 0.0 = rating2 wins
func calculateNewRating(rating1 int, rating2 int, score float64) (int, int) {
	k := 32.0
	expectedScore := calculateExpectedScore(rating1, rating2)

	new1 := rating1 + int(k*(score-expectedScore))
	new2 := rating2 + int(k*(expectedScore-score))
	return new1, new2
}

func loadSongs() Songs {
	fmt.Println("Loading songs...")

	// Open the sqlite3 database
	db, err := sql.Open("sqlite3", "db.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	// Query the database
	rows, err := db.Query("SELECT * FROM songs")
	if err != nil {
		fmt.Println(err)
	}

	// Iterate over the rows
	songs := make(map[string]Song)
	for rows.Next() {
		var song Song
		err = rows.Scan(&song.Song, &song.Artist, &song.Writer, &song.Album, &song.Year, &song.Rating)
		if err != nil {
			fmt.Println(err)
		}
		songs[song.Song] = song
	}

	return songs
}

func loadVoteCounter() int {
	var votes int

	fmt.Println("Loading vote counter...")

	// Open the sqlite3 database
	db, err := sql.Open("sqlite3", "db.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	// Query the database
	rows, err := db.Query("SELECT Count FROM counters WHERE Key = 'votes'")
	if err != nil {
		fmt.Println(err)
	}

	// Iterate over the rows
	for rows.Next() {
		err = rows.Scan(&votes)
		if err != nil {
			fmt.Println(err)
		}
	}

	return votes
}

func getPairOfSongs(songs Songs) (Song, Song) {
	rand1 := rand.Intn(len(songs))
	rand2 := rand.Intn(len(songs))
	for rand1 == rand2 {
		rand2 = rand.Intn(len(songs))
	}

	var song1 Song
	var song2 Song

	i := 0
	for _, song := range songs {
		if i == rand1 {
			song1 = song
		}
		if i == rand2 {
			song2 = song
		}
		i++
	}

	return song1, song2
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
	song1, song2 := getPairOfSongs(songs)

	htmlTemplate := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Vote on Taylor's Songs</title>
	</head>

	<style>
		html {
			font-family: Arial, sans-serif;
		}

		h1 {
			text-align: center;
		}

		.songs-container {
			display: flex;
			flex-wrap: wrap;
			justify-content: center;
			gap: 1em;
		}

		.song-card {
			border: 1px solid #ddd;
			padding: 8px;
			border-radius: 16px;
			box-shadow: 0 4px 8px 0 rgba(0,0,0,0.2);
			width: 400px;
		}

		.song-card h2 {
			margin: 0.5em 0;
		}

		.song-card button {
			background-color: #f4f4f4;
			border: none;
			border-radius: 100px;
			color: #000;
			padding: 10px 20px;
			text-align: center;
			border-radius: 100px;
			font-size: 1rem;
			font-weight: bold;
			color: white;
			background: #1b1bac;
			cursor: pointer;
		}

		.song-card button:hover {
			background-color: #3636eb;
		}

		.link-button {
			max-width: 600px;
			padding: 10px 20px;
			border-radius: 100px;
			text-align: center;
			background: #6700e6;
			font-weight: bold;
			text-decoration: none;
			cursor: pointer;
			color: white;
			margin: 2em auto;
			display: block;
		}

		.link-button:hover {
			background: #863ede;
		}
	</style>

	<body>
		<h1>Which song is better?</h1>
		<div class="songs-container">
			<div class="song-card">
				<h2>{{.Song1.Song}}</h2>
				<p>Artist(s): {{.Song1.Artist}}</p>
				<p>Writer(s): {{.Song1.Writer}}</p>
				<p>Album: {{.Song1.Album}}</p>
				<p>{{.Song1.Year}}</p>
				<button id="song1-button">Vote for {{.Song1.Song}}</button>
			</div>
			<div class="song-card">
				<h2>{{.Song2.Song}}</h2>
				<p>Artist(s): {{.Song2.Artist}}</p>
				<p>Writer(s): {{.Song2.Writer}}</p>
				<p>Album: {{.Song2.Album}}</p>
				<p>{{.Song2.Year}}</p>
				<button id="song2-button">Vote for {{.Song2.Song}}</button>
			</div>
		</div>
		<a class="link-button" href="/leaderboard">Leaderboard</a>
		<p style="max-width: 600px; margin: 1em auto;">
			You have cast <span id="votes-cast">0</span> votes.
		</p>
		<h2 style="max-width: 600px; margin: 1em auto;">How does this site work?</h2>
		<p style="max-width: 600px; margin: 1em auto;">
			The goal of this site is to rank <span style="font-style: italic;">all</span> of Taylor Swift's songs.
			We want to find out: What are her best songs according to Swifties? What are her worst?
		</p>
		<p style="max-width: 600px; margin: 1em auto;">
			Your job is to vote between two songs at a time.
			We use the Elo rating system to rank the songs, which is commonly used in chess, video games, and sports.
			If a song wins, its rating will increase, and if it loses, its rating will decrease.
		</p>
		<p style="max-width: 600px; margin: 1em auto;">
			You can vote as many times as you want.
			If you want your favorite song to win, vote a bunch of times and tell your friends!
		</p>
		<div style="height: 64px;"></div>
	</body>

	<script>
		let votesCast = localStorage.getItem("votesCast") ?? 0;
		let votesCastElement = document.getElementById("votes-cast")
		votesCastElement.textContent = votesCast;

		let song1Button = document.getElementById("song1-button");
		let song2Button = document.getElementById("song2-button");

		let song1Title = "{{.Song1.Song}}";
		let song2Title = "{{.Song2.Song}}";

		function vote(winner, loser) {
			votesCast++;
			localStorage.setItem("votesCast", votesCast);
			votesCastElement.textContent = votesCast;

			fetch("/vote", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify({winner, loser}),
			})
				.then(response => response.json())
				.then(data => {
					song1Title = data.song1.Song;
					song2Title = data.song2.Song;

					const songCards = document.querySelector(".song-card h2");
					const h2 = document.querySelectorAll(".song-card h2");
					const p1 = document.querySelectorAll(".song-card p:nth-of-type(1)");
					const p2 = document.querySelectorAll(".song-card p:nth-of-type(2)");
					const p3 = document.querySelectorAll(".song-card p:nth-of-type(3)");
					const p4 = document.querySelectorAll(".song-card p:nth-of-type(4)");
					const buttons = document.querySelectorAll(".song-card button");
					h2[0].textContent = data.song1.Song;
					p1[0].textContent = "Artist(s): " + data.song1.Artist;
					p2[0].textContent = "Writer(s): " + data.song1.Writer;
					p3[0].textContent = "Album: " + data.song1.Album;
					p4[0].textContent = data.song1.Year;
					buttons[0].textContent = "Vote for " + data.song1.Song;

					h2[1].textContent = data.song2.Song;
					p1[1].textContent = "Artist(s): " + data.song2.Artist;
					p2[1].textContent = "Writer(s): " + data.song2.Writer;
					p3[1].textContent = "Album: " + data.song2.Album;
					p4[1].textContent = data.song2.Year;
					buttons[1].textContent = "Vote for " + data.song2.Song;
				});
		}

		song1Button.addEventListener("click", function() {
			vote(song1Title, song2Title);
		});

		song2Button.addEventListener("click", function() {
			vote(song2Title, song1Title);
		});
	</script>

	</html>
	`

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, map[string]interface{}{
		"Song1": song1,
		"Song2": song2,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handlerVote(w http.ResponseWriter, r *http.Request) {
	body := map[string]interface{}{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	winner := body["winner"].(string)
	loser := body["loser"].(string)

	song1 := songs[winner]
	song2 := songs[loser]

	// Calculate new ratings
	new1, new2 := calculateNewRating(song1.Rating, song2.Rating, 1.0)

	// Update the ratings in memory
	song1.Rating = new1
	song2.Rating = new2
	songs[winner] = song1
	songs[loser] = song2

	// Open the sqlite3 database
	db, err := sql.Open("sqlite3", "db.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	// Update the ratings in the database
	_, err = db.Exec("UPDATE songs SET Rating = ? WHERE Song = ?", new1, winner)
	if err != nil {
		fmt.Println(err)
	}
	_, err = db.Exec("UPDATE songs SET Rating = ? WHERE Song = ?", new2, loser)
	if err != nil {
		fmt.Println(err)
	}

	// Increment votes in memory
	voteCounter++

	// Increment votes in counters table
	_, err = db.Exec("UPDATE counters SET Count = Count + 1 WHERE Key = 'votes'")
	if err != nil {
		fmt.Println(err)
	}

	// Return new pair of songs
	song1, song2 = getPairOfSongs(songs)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"song1": song1,
		"song2": song2,
	})
}

func handlerLeaderboard(w http.ResponseWriter, r *http.Request) {
	type RankedSong struct {
		Rank   int
		Song   string
		Rating int
	}
	rankedSongs := make([]RankedSong, 0, len(songs))
	for _, song := range songs {
		rankedSongs = append(rankedSongs, RankedSong{0, song.Song, song.Rating})
	}

	// Sort the songs by rating
	for i := 0; i < len(rankedSongs); i++ {
		for j := i + 1; j < len(rankedSongs); j++ {
			if rankedSongs[i].Rating < rankedSongs[j].Rating {
				rankedSongs[i], rankedSongs[j] = rankedSongs[j], rankedSongs[i]
			}
		}
	}

	// Add the rank to each song
	for i := 0; i < len(rankedSongs); i++ {
		rankedSongs[i].Rank = i + 1
	}

	htmlTemplate := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Leaderboard</title>
	</head>

	<style>
		html {
			font-family: Arial, sans-serif;
		}

		h1 {
			text-align: center;
		}

		.link-button {
			max-width: 600px;
			padding: 10px 20px;
			border-radius: 100px;
			text-align: center;
			background: #6700e6;
			font-weight: bold;
			text-decoration: none;
			cursor: pointer;
			color: white;
			margin: 2em auto;
			display: block;
		}

		.link-button:hover {
			background: #863ede;
		}

		table {
			margin: 0 auto;
			border-collapse: collapse;
		}

		th {
			text-align: left;
		}

		th, td {
			padding: 8px 16px 8px 8px;
		}

		tr:nth-child(even) {
			background-color: #eee;
		}
	</style>

	<body>
		<a class="link-button" href="/">Start Voting!</a>
		<h1>Leaderboard</h1>
		<p style="text-align: center;">
			Swifties have cast <span id="votes-cast">{{.voteCounter}}</span> votes total
		</p>
		<table>
			<tr>
				<th>Rank</th>
				<th>Song</th>
				<th>Rating</th>
			</tr>
			{{range $song := .rankedSongs}}
			<tr>
				<td>{{$song.Rank}}</td>
				<td>{{$song.Song}}</td>
				<td>{{$song.Rating}}</td>
			</tr>
			{{end}}
		</table>
		<div style="height: 64px;"></div>
	</body>
	</html>
	`

	tmpl, err := template.New("leaderboard").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, map[string]interface{}{
		"rankedSongs": rankedSongs,
		"voteCounter": voteCounter,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	songs = loadSongs()
	voteCounter = loadVoteCounter()

	http.HandleFunc("/", handlerRoot)
	http.HandleFunc("/vote", handlerVote)
	http.HandleFunc("/leaderboard", handlerLeaderboard)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		// Do nothing.
		// This is to prevent the server from logging requests twice.
	})
	port := ":8080"
	fmt.Printf("Starting server on port %s...\n", port)
	http.ListenAndServe(port, nil)
}
