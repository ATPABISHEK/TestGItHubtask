package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

//global variables which is accessed by all methods
var articleCollection *mongo.Collection
var ctx context.Context

//====================utitlity methods are writtern here==========================

//whenever we are handling error we will pass that error value to this methos
func errorHandler(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

//this method gets cursor to Collection of articles and send articles as response to request
func myResponseWriter(cursor *mongo.Cursor, w http.ResponseWriter) {
	var searchedArticles []bson.M
	err := cursor.All(ctx, &searchedArticles)
	errorHandler(err)
	var articleArr = make([]bson.M, len(searchedArticles))
	var articleMap = make(map[string]interface{}, 1)
	for index, article := range searchedArticles {
		articleArr[index] = article
	}
	articleMap["articles"] = articleArr
	finalRes, err := json.Marshal(articleMap)
	errorHandler(err)
	fmt.Fprintf(w, string(finalRes))

}

//it is called when route is "\articles" and request is POST

func articlePostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cursor, err := articleCollection.Find(ctx, bson.M{})
	errorHandler(err)
	var searchedArticles []bson.M
	err = cursor.All(ctx, &searchedArticles)
	errorHandler(err)
	count := len(searchedArticles)
	postResult, err := articleCollection.InsertOne(ctx, bson.M{
		"id":        fmt.Sprint(count),
		"title":     r.Form.Get("title"),
		"subtitle":  r.Form.Get("subtitle"),
		"content":   r.Form.Get("content"),
		"timestamp": time.Now().String(),
	})
	fmt.Println(postResult)
}

//====================http request handler(end point) methods are here=================

//it is called when route is "\"

func homePageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		fmt.Fprintf(w, "Hit on Home page as GET")
	case http.MethodPost:
		fmt.Fprintf(w, "Hit on Home page as POST")
	default:
		fmt.Fprintf(w, "Invalid request method only get and post")
	}
}

//it is called when route is "\articles" and request is GET if request is POST it calls articlePostHandler

func articleHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/article/")
	if len(id) > 0 {
		idHandler(w, r, id)
	} else {
		switch r.Method {
		case http.MethodGet:
			cursor, err := articleCollection.Find(ctx, bson.M{})
			errorHandler(err)
			myResponseWriter(cursor, w)
			//fmt.Println(articles)
		case http.MethodPost:
			articlePostHandler(w, r)
		default:
			fmt.Fprintf(w, "Error Request method is invalid")

		}
	}
}

//when route is "/article/search?q=key" then this function is called it searches for key in titlt,sutitle,content

func quarySearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		key := r.URL.Query().Get("q")
		if len(key) > 0 {
			fmt.Printf("query is %s", key)
			cursor, err := articleCollection.Find(ctx, bson.M{
				"$or": []interface{}{
					bson.M{"title": key},
					bson.M{"subtitle": key},
					bson.M{"content": key},
				},
			})
			errorHandler(err)
			myResponseWriter(cursor, w)
		} else {
			fmt.Fprintf(w, "There is no search result for this search key")
		}
	default:
		fmt.Fprintf(w, "Error Request method is invalid")
	}
}

//if route is "/article/id" then this function is called and it will response the article with given id

func idHandler(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		var idPat = regexp.MustCompile("[0-9]+")
		if idPat.MatchString(id) {
			fmt.Printf("%s", id)
			var article bson.M
			cursor := articleCollection.FindOne(ctx, bson.M{"id": id})
			err := cursor.Decode(&article)
			errorHandler(err)
			j, err := json.Marshal(article)
			errorHandler(err)
			fmt.Fprintf(w, string(j))
		} else {
			fmt.Fprintf(w, "The id is not found")
		}
	default:
		fmt.Fprintf(w, "Error Request method is invalid")
	}
}

//this function is for testing purpose it will post the hard coded article to Database
//it is called when route is "/article/post"
//this is dummy route for testing POST request only for development time test

func postRequester(w http.ResponseWriter, r *http.Request) {
	data := url.Values{
		"title":    {"The ATP"},
		"subtitle": {"MongoDb test"},
		"content":  {"this is testing mongoDb"},
	}
	resp, err := http.PostForm("http://localhost:8080/article/", data)
	errorHandler(err)
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	fmt.Println(resp)
}

//Here we handle all endpoint requests and making conection with mongoDB atlas database

func main() {

	//making connection with mongoDB data base
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://abi:abi123@articles.ixmpf.mongodb.net/<dbname>?retryWrites=true&w=majority"))
	errorHandler(err)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	errorHandler(err)
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	errorHandler(err)
	//getting collection from mongoDB database
	databaseHandler := client.Database("MyDatabase")
	articleCollection = databaseHandler.Collection("Articles")

	//here handling endpoint requests
	http.HandleFunc("/", homePageHandler)
	http.HandleFunc("/article/", articleHandler)
	http.HandleFunc("/article/search", quarySearchHandler)
	http.HandleFunc("/article/post", postRequester) //this is only for development time test purpose
	err = http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}
