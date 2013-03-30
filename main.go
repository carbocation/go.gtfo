package main

import (
	"fmt"
	"github.com/carbocation/forum.git/forum"
	"github.com/carbocation/util.git/datatypes/binarytree"
	"github.com/carbocation/util.git/datatypes/closuretable"
	"net/http"
	"strconv"
	"time"
	//"html/template"
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
)

var db *sql.DB

var store = sessions.NewCookieStore([]byte("something-very-secret"))

func main() {
	// Initialize the DB in the main function so we'll have a pool of connections maintained
	db = initdb()
	defer db.Close()

	//Initialize our router
	r := mux.NewRouter()

	//Create a subrouter for GET requests
	g := r.Methods("GET").Subrouter()
	g.HandleFunc("/", defaultHandler)
	g.HandleFunc("/thread/{id:[0-9]+}", threadHandler)
	g.HandleFunc("/css/{file}", cssHandler)
	g.HandleFunc("/hello/{name}", commentHandler)

	//Create a subrouter for POST requests
	p := r.Methods("POST").Subrouter()
	p.HandleFunc("/thread", newThreadHandler)
	p.HandleFunc("/login/{id:[0-9]+}", loginHandler)

	//Notify the http package about our router
	http.Handle("/", r)

	//Launch the server
	http.ListenAndServe("localhost:9999", nil)
}

func initdb() *sql.DB {
	db, err := sql.Open("postgres", "dbname=forumtest sslmode=disable")
	if err != nil {
		panic(err)
	}

	return db
}

func newThreadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Creating new threads is not yet implemented.\n")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "user")
	defer session.Save(r, w)

	session.Values["id"] = mux.Vars(r)["id"]
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	remPartOfURL := r.URL.Path[len("/"):]
	fmt.Fprintf(w, "<html><head><link rel=\"stylesheet\" href=\"/css/main.css\"></head><body><h1>Welcome, %s</h1><a href='/hello/'>Say hello</a>", remPartOfURL)

	fmt.Fprint(w, "</body></html>")
}

func cssHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")

	file := mux.Vars(r)["file"]

	switch {
	case file == "main.css":
		fmt.Fprintf(w, "%s", mainCss())
	}

}

func mainCss() string {
	return `
div .comment {
	padding-left: 100px;
}
`
}

func commentHandler(w http.ResponseWriter, r *http.Request) {
	remPartOfURL := r.URL.Path[len("/hello/"):] //get everything after the /hello/ part of the URL
	//w.Header().Set("Content-Type", "text/html")

	fmt.Fprint(w, "<html><head><link rel=\"stylesheet\" href=\"/css/main.css\"></head><body>")
	fmt.Fprintf(w, "Hello %s!", remPartOfURL)

	PrintNestedComments(w, ClosureTree())

	fmt.Fprint(w, "</body></html>")
}

func threadHandler(w http.ResponseWriter, r *http.Request) {
	unsafeId := r.URL.Path[len("/thread/"):]

	//If the thread ID is not parseable as an integer, stop immediately
	id, err := strconv.ParseInt(unsafeId, 10, 64)
	if err != nil {
		return
	}

	// Generate a closuretable from the root requested id
	ct := closuretable.New(id)
	// Pull down the remaining elements in the closure table that are descendants of this node
	q := `select * 
from entry_closures
where descendant in (
select descendant
from entry_closures
where ancestor=$1
)
and ancestor in (
select descendant
from entry_closures
where ancestor=$1
)
and depth = 1`
	stmt, err := db.Prepare(q)
	if err != nil {
		//fmt.Printf("Statement Preparation Error: %s", err)
		return
	}

	rows, err := stmt.Query(unsafeId)
	if err != nil {
		//fmt.Printf("Query Error: %v", err)
		return
	}

	//Populate the closuretable
	for rows.Next() {
		var ancestor, descendant int64
		var depth int
		err = rows.Scan(&ancestor, &descendant, &depth)
		if err != nil {
			//fmt.Printf("Rowscan error: %s", err)
			return
		}

		err = ct.AddChild(closuretable.Child{Parent: ancestor, Child: descendant})

		//err = ct.AddRelationship(closuretable.Relationship{Ancestor: ancestor, Descendant: descendant, Depth: depth})
		if err != nil {
			//fmt.Fprintf(w, "Error: %s", err)
			return
		}
	}

	id, entries, err := forum.RetrieveDescendantEntries(unsafeId, db)
	if err != nil {
		//fmt.Fprintf(w, "Error: %s", err)
		return
	}

	//fmt.Printf("Entries: %#v, %s", entries, err)

	//Obligatory boxing step
	interfaceEntries := map[int64]interface{}{}
	for k, v := range entries {
		interfaceEntries[k] = v
	}

	tree, err := ct.TableToTree(interfaceEntries)
	if err != nil {
		//fmt.Printf("TableToTree error: %s", err)
		return
	}

	fmt.Fprint(w, "<html><head><link rel=\"stylesheet\" href=\"/css/main.css\"></head><body>")
	PrintNestedComments(w, tree)
	fmt.Fprint(w, "</body></html>")
}

func PrintNestedComments(w http.ResponseWriter, el *binarytree.Tree) {
	if el == nil {
		return
	}

	fmt.Fprint(w, "<div class=\"comment\">")
	//Self
	e := el.Value.(forum.Entry)
	fmt.Fprintf(w, "Title: %s", e.Title)

	//Children are nested
	PrintNestedComments(w, el.Left())
	fmt.Fprint(w, "</div>")

	//Siblings are parallel
	PrintNestedComments(w, el.Right())
}

func ClosureTree() *binarytree.Tree {
	//Make some entries
	entries := map[int64]forum.Entry{
		0: forum.Entry{Id: 100, Title: "Title 100", Body: "Body 100", Created: time.Now(), AuthorId: 0},
		1: forum.Entry{Id: 101, Title: "Title 101", Body: "Body 101", Created: time.Now(), AuthorId: 1},
		2: forum.Entry{Id: 102, Title: "Title 102", Body: "Body 102", Created: time.Now(), AuthorId: 2},
		3: forum.Entry{Id: 103, Title: "Title 103", Body: "Body 103", Created: time.Now(), AuthorId: 3},
	}

	ct := closuretable.New(0)
	ct.AddChild(closuretable.Child{Parent: 0, Child: 1})
	ct.AddChild(closuretable.Child{Parent: 0, Child: 2})
	ct.AddChild(closuretable.Child{Parent: 1, Child: 3})

	// Obligatory boxing step
	// Convert to interface type so the generic TableToTree method can be called on these entries
	boxedEntries := map[int64]interface{}{}
	for k, v := range entries {
		boxedEntries[k] = v
	}

	tree, _ := ct.TableToTree(boxedEntries)

	return tree
}
