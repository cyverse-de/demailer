package main

import (
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"

	"github.com/cyverse-de/configurate"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

// RootTemplateName is the, uh, name of the root template. We're using ParseFiles
// to parse all of the templates at once and associate them with a root template.
const RootTemplateName = "emails"

// Demailer contains the logic for the application.
type Demailer struct {
	config      *viper.Viper
	templateDir string
	router      *mux.Router
	t           *template.Template
}

// ListTemplates is the API handler that prints out a list of template names in
// the format:
//   {
//     "templates" : [ "template name" ... ]
//   }
func (d *Demailer) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := map[string][]string{
		"templates": []string{},
	}

	for _, tmpl := range d.t.Templates() {
		if tmpl.Name() != RootTemplateName {
			templates["templates"] = append(templates["templates"], tmpl.Name())
		}
	}

	j, err := json.Marshal(templates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(j))
}

// InitRoutes sets up the API routes for the Demailer instance.
func (d *Demailer) InitRoutes() error {
	fs := http.FileServer(http.Dir(d.templateDir))

	d.router.Handle("/debug/vars", http.DefaultServeMux)
	d.router.PathPrefix("/templates/").Handler(http.StripPrefix("/templates/", fs))
	d.router.HandleFunc("/templates", d.ListTemplates).Methods("GET")
	return nil
}

// Init sets up the Demailer instance by parsing the template files and
// associating them with it.
func (d *Demailer) Init() error {
	files, err := ioutil.ReadDir(d.templateDir)
	if err != nil {
		return err
	}
	var filepaths []string
	for _, file := range files {
		filepaths = append(filepaths, path.Join(d.templateDir, file.Name()))
	}

	d.t, err = template.New(RootTemplateName).ParseFiles(filepaths...)
	if err != nil {
		return err
	}

	return d.InitRoutes()
}

func main() {
	var (
		configPath  = flag.String("config", "/etc/jobservices.yml", "Path to the configuration file")
		templateDir = flag.String("templates", "./templates", "Directory path containing email templates")
		addr        = flag.String("addr", ":60000", "API listen port")
	)

	cfg, err := configurate.InitDefaults(*configPath, configurate.JobServicesDefaults)
	if err != nil {
		log.Fatal(err)
	}

	app := &Demailer{
		config:      cfg,
		templateDir: *templateDir,
		router:      mux.NewRouter(),
	}

	app.Init()

	log.Fatal(http.ListenAndServe(*addr, app.router))
}
