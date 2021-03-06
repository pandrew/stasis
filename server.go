package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	//"github.com/pandrew/stasis/drivers"
)

const (
	extPreinstall string = ".preinstall"
	extGohtml     string = ".gohtml"
	extInstall    string = ".install"
)

func GetStasisDir() string {
	return fmt.Sprintf(filepath.Join(GetHomeDir(), ".stasis"))
}

func hostDir() string {
	return filepath.Join(GetStasisDir(), "machines")
}

func preinstallDir() string {
	return filepath.Join(GetStasisDir(), "preinstall")
}

func gohtmlDir() string {
	return filepath.Join(GetStasisDir(), "gohtml")
}

func installDir() string {
	return filepath.Join(GetStasisDir(), "install")
}

func postinstallDir() string {
	return filepath.Join(GetStasisDir(), "postinstall")
}

func staticDir() string {
	return filepath.Join(GetStasisDir(), "static")
}

func DirExists(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func init() {
	dirInstall := installDir()
	pathExist, _ := DirExists(dirInstall)
	if !pathExist {
		if err := os.MkdirAll(dirInstall, 0700); err != nil {
			log.Println(err)
		}
	}
	pathPreinstall := preinstallDir()
	pathPreinstallExist, _ := DirExists(pathPreinstall)
	if !pathPreinstallExist {
		dirPreinstall := preinstallDir()
		uri := "https://github.com/pandrew/stasis-preinstall.git"

		err := gitDownload(dirPreinstall, uri)
		if err != nil {
			log.Fatal(os.Stderr, err)
		}
	}
}

func initRouter() {
	r := mux.NewRouter()
	// Prepend uri with v1 for version 1 api. This will help error responds
	// when using relative paths in links.
	r.HandleFunc("/v1/{id}/inspect", ReturnInspect)
	r.HandleFunc("/v1/{id}/preinstall", ReturnPreinstall)
	r.HandleFunc("/v1/{id}/preinstall/raw", ReturnRawPreinstall)
	r.HandleFunc("/v1/{id}/preinstall/preview", ReturnPreviewPreinstall)
	r.HandleFunc("/v1/{id}/preinstall/toggle", TogglePreinstall)
	r.HandleFunc("/v1/{id}/preinstall/disable", DisablePreinstall)
	r.HandleFunc("/v1/{id}/preinstall/enable", EnablePreinstall)
	r.HandleFunc("/v1/{id}/install", ReturnInstall)
	r.HandleFunc("/v1/{id}/install/raw", ReturnRawInstall)
	r.HandleFunc("/v1/{id}/install/toggle", ToggleInstall)
	r.HandleFunc("/v1/{id}/status/toggle", Toggle)
	r.HandleFunc("/v1/{id}/preinstall/disable", DisablePreinstall)
	r.HandleFunc("/v1/info/stats", ReturnStats)
	r.HandleFunc("/v1/{id}/announce", GatherMac)
	r.HandleFunc("/v1/{id}/select", Select)
	http.Handle("/", r)

	port := os.Getenv("STASIS_HTTP_PORT")
	if port == "" {
		os.Setenv("STASIS_HTTP_PORT", "8080")
	}
	addr := os.Getenv("STASIS_HTTP_ADDR")
	if addr == "" {
		os.Setenv("STASIS_HTTP_ADDR", "localhost")
	}
	log.Info("Listening on: ", os.Getenv("STASIS_HTTP_ADDR")+":"+os.Getenv("STASIS_HTTP_PORT"))
	path := os.Getenv("STASIS_HOST_STORAGE_PATH")
	log.Info("Using path: ", path)
	static := os.Getenv("STASIS_HTTP_STATIC_PATH")
	log.Info("Using static path: ", static)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(static)))

	log.Println("Listening...")
	http.ListenAndServe(os.Getenv("STASIS_HTTP_ADDR")+":"+os.Getenv("STASIS_HTTP_PORT"), nil)
}

func ReturnInspect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macaddress := vars["id"]

	_, err := ValidateMacaddr(macaddress)
	if err != nil {
		http.NotFound(w, r)
	} else {
		store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
		host, err := store.GetMacaddress(macaddress)
		if err != nil {
			log.Fatal(err)
		}
		prettyJSON, err := json.MarshalIndent(host, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		//log.Println(getHost(c))
		fmt.Fprintf(w, string(prettyJSON))
	}
}

func ReturnStats(w http.ResponseWriter, r *http.Request) {
	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))

	hostList, err := store.List()
	if err != nil {
		log.Fatal(err)
	}

	items := []hostListItem{}
	hostListItems := make(chan hostListItem)

	for _, host := range hostList {
		go getHostState(host, *store, hostListItems)
	}

	for i := 0; i < len(hostList); i++ {
		items = append(items, <-hostListItems)
	}

	close(hostListItems)
	templates, err := template.New("stats").Parse(index)
	if err != nil {
		panic(err)
	}
	err = templates.Execute(w, items)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

func ReturnInstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macaddress := vars["id"]

	_, err := ValidateMacaddr(macaddress)
	if err != nil {
		http.NotFound(w, r)
	} else {

		store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
		host, err := store.GetMacaddress(macaddress)
		if err != nil {
			log.Fatal(err)
		}
		inst := installDir()
		ValidateTemplates(inst, extInstall)
		//test := host.Install
		if len(host.Install) != 0 && host.PermitInstall == true {
			tmpl := host.Install + extInstall
			renderTemplate(w, tmpl, host)
			host.PermitInstall = false
		} else {
			http.NotFound(w, r)

		}

	}

}

func ReturnRawInstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macaddress := vars["id"]

	_, err := ValidateMacaddr(macaddress)
	if err != nil {
		http.NotFound(w, r)
	} else {

		store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
		host, err := store.GetMacaddress(macaddress)
		if err != nil {
			log.Fatal(err)
		}
		if len(host.Install) != 0 {
			dir := installDir()
			returnRaw(w, dir, host.Install, extInstall)
		} else {
			http.NotFound(w, r)
		}
	}

}

func ReturnPreinstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macaddress := vars["id"]
	if macaddress == "" {
		http.NotFound(w, r)
		return
	}

	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	host, err := store.GetMacaddress(macaddress)
	if err != nil {
		log.Fatal(err)
	}

	active, err := store.GetActive()
	if err != nil {
		log.Println(err)
	}

	if host.Name == active.Name {
		if host.PermitPreinstall == true {
			pre := preinstallDir()
			ValidateTemplates(pre, extPreinstall)

			tmpl := host.Preinstall + extPreinstall
			renderTemplate(w, tmpl, host)

			host.PermitPreinstall = false
			host.Status = "INSTALLED"
			host.SaveConfig()
		} else if host.Status == "INSTALLED" {
			ip := GetIP(r)
			log.Errorf("%s requests %s: host is already installed!", ip, macaddress)
		} else {
			ip := GetIP(r)
			log.Errorf("%s requests %s: not in database!", ip, macaddress)
		}
	} else {
		ip := GetIP(r)
		log.Errorf("%s requests %s: Host does not match selected!", ip, macaddress)

	}
}

func ReturnRawPreinstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	match := ValidateHostName(id)
	if match == false {
		http.NotFound(w, r)
	} else {
		store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
		host, err := store.GetHostname(id)
		if err != nil {
			log.Println(err)
		}
		dir := preinstallDir()
		returnRaw(w, dir, host.Preinstall, extPreinstall)
	}
}

func ReturnPreviewPreinstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	match := ValidateHostName(id)
	if match == false {
		http.NotFound(w, r)
	} else {

		store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
		host, err := store.GetHostname(id)
		if err != nil {
			log.Println(err)
		}
		pre := preinstallDir()
		ValidateTemplates(pre, extPreinstall)
		tmpl := host.Preinstall + extPreinstall
		renderTemplate(w, tmpl, host)
	}
}

func GatherMac(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macaddress := vars["id"]
	if macaddress == "" {
		http.NotFound(w, r)
		return
	}

	ValidateMacaddr(macaddress)

	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	// Locate the host
	host, err := store.GetActive()
	if err != nil {
		log.Println(err)
	}

	ip := GetIP(r)

	if macaddress == host.Macaddress {
		http.NotFound(w, r)
		log.Errorf("Source %s requests to modify host %q to source macaddress %q: DENIED", ip, host.Name, macaddress)
		return
	} else {
		log.Printf("Source %s requests to modify host %q to source macaddress %q: ACCEPTED", ip, host.Name, macaddress)
		host.Macaddress = macaddress
		host.SaveConfig()
	}
}

var templates *template.Template

func Toggle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname := vars["id"]
	//macaddress := vars["id"]
	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	host, err := store.Load(hostname)
	if err != nil {
		log.Println(err)
	}

	log.Println(host)
	if host.Announce == false {
		host.Announce = true
		host.PermitPreinstall = true
		host.PermitInstall = true
		host.Status = "ACTIVE"
		log.Infof("%s: Announce is now true", host.Name)
	} else if host.Announce == true {
		host.Announce = false
		host.PermitPreinstall = false
		host.PermitInstall = false
		host.Status = "INACTIVE"
		log.Infof("%s: Announce is now false", host.Name)
	} else {
		host.Announce = false
		host.PermitPreinstall = false
		host.PermitInstall = false
		host.Status = "INSTALLED"
		log.Infof("%s is now INSTALLED", host.Name)

	}

	host.SaveConfig()
	http.Redirect(w, r, "/v1/info/stats", http.StatusFound)

}

func ToggleInstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname := vars["id"]
	//macaddress := vars["id"]
	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	host, err := store.Load(hostname)
	if err != nil {
		log.Println(err)
	}

	log.Println(host)
	if host.PermitInstall == false {
		host.PermitInstall = true
		log.Infof("%s: PermitInstall is now true", host.Name)
	} else if host.PermitInstall == true {
		host.PermitInstall = false
		log.Infof("%s: PermitInstall is now false", host.Name)
	}

	host.SaveConfig()
	http.Redirect(w, r, "/v1/info/stats", http.StatusFound)
}

func TogglePreinstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname := vars["id"]
	//macaddress := vars["id"]
	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	host, err := store.Load(hostname)
	if err != nil {
		log.Println(err)
	}

	log.Println(host)
	if host.PermitPreinstall == false {
		host.PermitPreinstall = true
		log.Infof("%s: PermitPreinstall is now true", host.Name)
	} else if host.PermitPreinstall == true {
		host.PermitPreinstall = false
		log.Infof("%s: PermitPreinstall is now false", host.Name)
	}

	host.SaveConfig()
	http.Redirect(w, r, "/v1/info/stats", http.StatusFound)
}

func DisablePreinstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname := vars["id"]
	//macaddress := vars["id"]
	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	host, err := store.Load(hostname)
	if err != nil {
		log.Println(err)
	}

	log.Println(host)
	if host.PermitPreinstall == true {
		host.PermitPreinstall = false
	}

	host.SaveConfig()
	http.Redirect(w, r, "/v1/info/stats", http.StatusFound)

}

func EnablePreinstall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname := vars["id"]
	//macaddress := vars["id"]
	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	host, err := store.Load(hostname)
	if err != nil {
		log.Println(err)
	}

	log.Println(host)
	if host.PermitPreinstall == false {
		host.PermitPreinstall = true
	}

	host.SaveConfig()
	http.Redirect(w, r, "/v1/info/stats", http.StatusFound)

}

func Select(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname := vars["id"]
	store := NewHostStore(os.Getenv("STASIS_HOST_STORAGE_PATH"))
	host, _ := store.Load(hostname)

	host, err := store.Load(hostname)
	if err != nil {
		log.Println(host)
	}

	store.SetActive(host)

	http.Redirect(w, r, "/v1/info/stats", http.StatusFound)

}

func renderTemplate(w http.ResponseWriter, tmpl string, vars interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func returnRaw(w http.ResponseWriter, dir string, tmpl string, ext string) {
	raw, err := ioutil.ReadFile(dir + "/" + tmpl + ext)
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(w, string(raw))
}

func GetIP(r *http.Request) string {
	if ipProxy := r.Header.Get("X-FORWARDED-FOR"); len(ipProxy) > 0 {
		return ipProxy
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
