package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"net/http"
	"log"
	"html/template"
	"path/filepath"
	"io/ioutil"
	"github.com/gorilla/mux"
	"strconv"
	"encoding/json"
	"encoding/xml"
)

type Metadata struct {
	Author string	`json:"author"`
	Title  string	`json:"title"`
	CTSurn string	`json:"ctsurn"`
}

type work struct {
	Metadata map[string]string	`json:"metadata"`
	Text []logicalunit			`json:"text"`
	InvaildUrns []int			`json:"invalidurns"`
}

type xmlnode struct {
	XMLName xml.Name
	N xml.Attr 					`xml:"n,attr"`
	Attrs []xml.Attr			`xml:",attr"`
	Text    string				`xml:",innerxml"`
	Subnodes []xmlnode			`xml:",any"`
}

type logicalunit struct {
	Name	string					`json:"name"`
	Urn     string					`json:"urn"`
	Text    string					`json:"text" xml:",innerxml"`
	Attributes map[string]string	`json:"attributes"`
	Subunits []logicalunit			`json:"subunits" xml:",any"`
}

type file struct {
	name string
	xml bool
	cex bool
	path string
}
//array to hold file structs representing files staged to be ingested into the corpus builder
var Newfiles []file
// array to hold the file structs representing the files that are already ingested into the corpus builder
var Files []file
//reads the data from a csv file
func loadCSV(file string,) (records [][]string, err error) {
	f, err := os.Open(file)//place holder
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	return r.ReadAll()
}
//parses a two column csv file (column1: urn, column2: text) into the the logical unit format of the corpus builder
func parseCSV(data [][]string) ([]logicalunit, []int) {
	var text []logicalunit
	var errors []int
	if checkforheads(data){
		for index, element := range data {
			if index == 0 {
				continue
			}
			var node logicalunit
			urn := element[0]
			if isCTSURN(urn){
				node = logicalunit{Urn:element[0],Text:element[1]}
			} else {
				fmt.Println("Your urn for this unit is not valid please input a valid urn")
				errors = append(errors, index)
				node = logicalunit{Urn:element[0],Text:element[1]}
			}
			text = append(text, node)
		}
	} else {
		for index, element := range data {
			var node logicalunit
			urn := element[0]
			if isCTSURN(urn){
				node = logicalunit{Urn:element[0],Text:element[1]}
			} else {
				fmt.Println("Your urn for this unit is not valid please input a valid urn")
				errors = append(errors, index)
				node = logicalunit{Urn:element[0],Text:element[1]}
			}
			text = append(text, node)
		}
	}
	return text, errors
}



// From citemicroservices version: 1.0.4 by https://github.com/ThomasK81
func isCTSURN(s string) bool {
	test := strings.Split(s, ":")
	switch {
	case len(test) < 4:
		return false
	case len(test) > 5:
		return false
	case test[0] != "urn":
		return false
	case test[1] != "cts":
		return false
	default:
		return true
	}
}
//helper function for checking if a csv file has heads that must be ignored during parsing
func checkforheads(file [][]string) (bool) {
	if file[0][0] == "identifier" && file[0][1] == "text" {
		return true
	} else {
		return false
	}
}
//checks what files are already ingested into the corpus builder and adds a file struct for each file into the files array
func loadcorpus(){
	oldtoast, err := ioutil.ReadDir("corpus/data")
	if err != nil {
		fmt.Printf("Failed to get the toast")
		log.Fatal(err)
	}
	for _, slice := range oldtoast {
		fmt.Println(slice.Name())
		var butter file
		butter = file{name:strings.TrimSuffix(slice.Name(), filepath.Ext(slice.Name())),xml:false,cex:false,path:"corpus/data/"+slice.Name()}
		Files = append(Files, butter)
	}
}
//looks in the input directory for file that should be staged for the corpus builder and new file object into the newfiles array
func loadnewfiles()  {
	newtoast, err := ioutil.ReadDir("input")
	if err != nil {
		fmt.Printf("Failed to get the toast")
		log.Fatal(err)
	}
	for _, slice := range newtoast {
		fmt.Println(slice.Name())
		var butter file
		butter = file{name:strings.TrimSuffix(slice.Name(), filepath.Ext(slice.Name())),xml:false,cex:false,path:"input/"+slice.Name()}
		Newfiles = append(Newfiles, butter)
	}
}
//files that are already ingested into the corpus builder. It returns these results in the form of two maps of string keys (file name) and value true.
func listfiles() (map[string]bool,map[string]bool){
	var goulds map [string] bool
	var firstPrimes map [string] bool
	goulds = make(map[string]bool)
	firstPrimes = make(map[string]bool)
	for _, gould := range Files  {
		goulds[gould.name] = true
	}
	for _, prime := range Newfiles {
		if goulds[prime.name] {
			continue
		} else {
			firstPrimes[prime.name] = true
		}
	}
	return goulds, firstPrimes
}
//saves in the logicalunit format to a json file
func savework(updatedwork work)  {
	bytes, _ := json.MarshalIndent(updatedwork, "", "\t")
	ioutil.WriteFile("corpus/data/"+updatedwork.Metadata["author"]+"_"+updatedwork.Metadata["title"]+".json",bytes, 0644)
	Files = append(Files, file{path:"corpus/data/"+updatedwork.Metadata["author"]+"_"+updatedwork.Metadata["title"]+".json", name:updatedwork.Metadata["author"]+"_"+updatedwork.Metadata["title"], xml:false, cex:false})
}
//take an xml node struct and the recursively formats it into logicalunits
func convertxmlnodetologicalunit(textnode xmlnode, urn string) []logicalunit {
	var ludata []logicalunit
	for _, element := range textnode.Subnodes {
		var node logicalunit
		node.Urn = urn + "." + element.N.Value
		node.Name = element.XMLName.Local
		if len(element.Subnodes) != 0 {
			node.Subunits = convertxmlnodetologicalunit(element, node.Urn)
		} else {
			node.Text = element.Text
		}
		ludata = append(ludata, node)
	}
	return ludata
}
// main function that starts router that handles users access with the server opens at localhost port 8000
func main() {
	loadcorpus()
	loadnewfiles()
	router := mux.NewRouter().StrictSlash(true)
	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	router.PathPrefix("/static/").Handler(s)
	//fs := http.FileServer(http.Dir("static"))
	//http.Handle("/static/", http.StripPrefix("/static/", fs))
	router.HandleFunc("/", handler)
	router.HandleFunc("/corpusfile/{nameoffile}", corpushandler)
	router.HandleFunc("/corpusfile/{nameoffile}/save", savehandler)
	router.HandleFunc("/staged/{nameoffile}", stagedhandler)
	router.HandleFunc("/staged/{nameoffile}/submit", submithandler)
	router.HandleFunc("/createcex/{nameofcorpus}", cexcorpus)
	log.Println("Listening at localhost:8000...")
	log.Fatal(http.ListenAndServe("localhost:8000", router))
	//log.Fatal(http.ListenAndServe("localhost:8000", nil))
}
//basic router handler that returns the home page for the user interface for the service
func handler(w http.ResponseWriter, r *http.Request){
	var data struct{
		A map[string]bool
		B map[string]bool
		C template.HTML
	}
	data.A, data.B = listfiles()
	data.C = "Select A Text From The Left To Edit An Existing File In Your Corpus or Add A New Staged File To Your Corpus"
	lp := filepath.Join("templates", "home.html")
	t, _ := template.ParseFiles(lp)
	t.Execute(w, data)
	//s := "test"
	//fmt.Fprintf(w, "url.path = %q\n", s)
}
//handler for loading a file into the the user interface that has already been ingested into the corpus builder
func corpushandler(w http.ResponseWriter, r *http.Request)  {
	var data struct{
		A map[string]bool
		B map[string]bool
		C map[string]string
		D []logicalunit
		E []int
	}
	vars := mux.Vars(r)
	var deathglider file
	for _, jaffa := range Files {
		if jaffa.name == vars["nameoffile"] {
			deathglider = jaffa
			break
		}
	}
	data.A, data.B = listfiles()
	file, e := ioutil.ReadFile(deathglider.path)
	if e != nil {fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(file))
	var tollon work
	json.Unmarshal(file, &tollon)
	fmt.Printf("Results: %v\n", tollon)
	data.C = tollon.Metadata
	data.D = tollon.Text
	data.E = tollon.InvaildUrns
	lp := filepath.Join("templates", "livefile.html")
	t, _ := template.ParseFiles(lp)
	t.Execute(w, data)
}
//handler that saves the current files being worked in the user interface.
func savehandler(w http.ResponseWriter, r *http.Request)  {
	var submiteddata []logicalunit
	if err := r.ParseForm(); err != nil {
		// handle error
	}
	for x := 0; x > -1 ; x++ {
		if _, ok := r.PostForm["urn" + strconv.Itoa(x)]; ok {
			submiteddata = append(submiteddata, logicalunit{Urn:r.PostForm[("urn" + strconv.Itoa(x))][0],Text:r.PostForm[("text" + strconv.Itoa(x))][0]})
		} else {
			x = -3
		}
	}
	var meta = make(map[string]string)
	meta["author"] = r.PostForm["author"][0]
	meta["title"] = r.PostForm["title"][0]
	savework(work{Text:submiteddata,Metadata:meta})
}
// handler that loads a file that has been staged in the input directory into the userinterface to be ingested into the corpus builder
func stagedhandler(w http.ResponseWriter, r *http.Request)  {
	var data struct{
		A map[string]bool
		B map[string]bool
		C map[string]string
		D []logicalunit
		E []int
	}
	vars := mux.Vars(r)
	data.A, data.B = listfiles()
	var deathglider file
	for _, jaffa := range Newfiles {
		if jaffa.name == vars["nameoffile"] {
			deathglider = jaffa
			break
		}
	}
	//Parse an XML file into the LU structure
	if string(deathglider.path[len(deathglider.path)-4:]) == ".xml" {
		file, e := ioutil.ReadFile(deathglider.path)
		if e != nil {fmt.Printf("File error: %v\n", e)
			os.Exit(1)
		}
		var tollon xmlnode
		xml.Unmarshal(file, &tollon)
		var errors []int

		//extracts the meta data from the xml file
		var lumeta = make(map[string]string)
		lumeta["author"] = ""
		lumeta["title"] = tollon.Subnodes[0].Subnodes[0].Subnodes[0].Subnodes[0].Text
		lumeta["publisher"] = tollon.Subnodes[0].Subnodes[0].Subnodes[1].Subnodes[0].Text
		//puts the metadata into the into the response objects
		data.C = lumeta

		//sends the textnode to the xml to LU conversion functions
		textnode := tollon.Subnodes[1].Subnodes[0].Subnodes[0]
		data.D = convertxmlnodetologicalunit(textnode, textnode.N.Value)
		out := data.D[0].Subunits[0].Subunits[0].Subunits[0]
		fmt.Printf(out.Urn)
		data.E = errors
		lp := filepath.Join("templates", "stagedxml.html")
		t, _ := template.ParseFiles(lp)
		t.Execute(w, data)
	} else if string(deathglider.path[len(deathglider.path)-4:]) == ".csv"{
		var tealc []logicalunit
		var errors []int
		var csvdata [][]string
		csvdata, _ = loadCSV(deathglider.path)
		tealc, _ = parseCSV(csvdata)
		var knox work
		var meta = make(map[string]string)
		meta["author"] = "unknown"
		meta["title"] = vars["nameoffile"]
		meta["ctsurn"] = "unknown"
		knox = work{meta,tealc,errors}
		data.C = knox.Metadata
		data.D = knox.Text
		data.E = knox.InvaildUrns
		lp := filepath.Join("templates", "staged.html")
		t, _ := template.ParseFiles(lp)
		t.Execute(w, data)
	} else {
		http.Error(w, "FAILURE: Unsupported file type please confirm that file is one of the types supported by the corpus builder. If it is a supported open issue on the github page so the bug can fixed. If it is not supported please open an issue on github requesting that the file format be added", 500)
	}

}
//handler that ingests a staged file that has been loaded into the user interface
func submithandler(w http.ResponseWriter, r *http.Request)  {
	var submiteddata []logicalunit
	if err := r.ParseForm(); err != nil {
		// handle error
	}
	for x := 0; x > -1 ; x++ {
		if _, ok := r.PostForm["urn" + strconv.Itoa(x)]; ok {
			submiteddata = append(submiteddata, logicalunit{Urn:r.PostForm[("urn" + strconv.Itoa(x))][0],Text:r.PostForm[("text" + strconv.Itoa(x))][0]})
		} else {
			x = -3
		}
	}
	var meta = make(map[string]string)
	meta["author"] = r.PostForm["author"][0]
	meta["title"] = r.PostForm["title"][0]
	meta["ctsurn"] = r.PostForm["ctsurn"][0]
	savework(work{Text:submiteddata,Metadata:meta})
	/*for key, values := range r.PostForm {
		// [...]
	}*/
}
//generates a cex corpus from the file ingested into the corpus builder
func cexcorpus(w http.ResponseWriter, r *http.Request)  {
	vars := mux.Vars(r)
	tablet, err := os.Create("corpus/CEX/"+vars["nameofcorpus"]+".cex")
	if err != nil {fmt.Printf("File error 1: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tablet.Close()
	var ctscatalog string
	for _, ori := range Files {
		file, e := ioutil.ReadFile(ori.path)
		if e != nil {fmt.Printf("File error: %v\n", e)
			os.Exit(1)
		}
		var priar work
		json.Unmarshal(file, &priar)
		follower := priar.Metadata
		ctscatalog = ctscatalog + follower["ctsurn"] + "\n"
	}
	tablet.WriteString("")
	tablet.WriteString("# CEX corpus created by Leipzig University corpus builder\n")
	tablet.WriteString("\n#!cexversion\n")
	tablet.WriteString("2.0\n")
	tablet.WriteString("\n#!citelibrary\n")
	tablet.WriteString("name#cexcorpus\n")
	tablet.WriteString("\n#!ctscatalog\n")
	tablet.WriteString("urn#citationScheme#groupName#workTitle#versionLabel#exemplarLabel#online#lang\n")
	tablet.WriteString(ctscatalog)
	tablet.WriteString("\n#!ctsdata\n")
	for _, ancient := range Files {
		file, e := ioutil.ReadFile(ancient.path)
		if e != nil {fmt.Printf("File error: %v\n", e)
			os.Exit(1)
		}
		var stargate work
		json.Unmarshal(file, &stargate)
		pegesasus := stargate.Text
		for _, asgardians := range pegesasus {
			tablet.WriteString(asgardians.Urn+"#  "+asgardians.Text+"\n")
		}
	}
}