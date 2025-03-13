package thoughts

import (
	"net/http"
)

type DatabaseAccess struct {
}

func Routes(mux *http.ServeMux) {
	// these are the two big chunks of stateful/configured functionality
	images := ImageProcessor{
		// TODO needs to
	}
	var queries DatabaseAccess

	admin := AdminController{
		// TODO needs to check session validity
		// TODO needs to insert notes into DB
		// TODO needs to insert images into DB
		Queries: queries,

		// TODO needs to process uploaded images
		// TODO needs to process downloaded images
		Images: images,
	}
	mux.HandleFunc("GET /admin", admin.AdminPage)
	mux.HandleFunc("POST /admin/new", admin.NewNote)
	mux.HandleFunc("POST /admin/upload", admin.UploadImage)
	mux.HandleFunc("POST /admin/download", admin.DownloadImage)

	assets := AssetController{
		// TODO needs to know where image thumbnails are
		Images: images,
	}
	assets.HandleAssets(mux)

	auth := AuthController{
		// TODO needs to check session validity
		// TODO needs to create and read passkeys (passkey impl doesn't need state)
		// TODO needs to create and read challenges
		Queries: queries,
	}
	mux.HandleFunc("GET /register", auth.RegisterPage)
	mux.HandleFunc("POST /register/start", auth.RegisterStart)
	mux.HandleFunc("POST /register/finish", auth.RegisterFinish)
	mux.HandleFunc("GET /login", auth.LoginPage)
	mux.HandleFunc("GET /login/start", auth.LoginStart)
	mux.HandleFunc("GET /login/finish", auth.LoginFinish)

	feeds := FeedController{
		// TODO needs to read notes
		Queries: queries,
	}
	mux.HandleFunc("GET /{$}", feeds.HomePage)
	mux.HandleFunc("GET /atom.xml", feeds.AtomFeed)
	mux.HandleFunc("GET /notes/{start}", feeds.WeekPage)
	mux.HandleFunc("GET /note/{id}", feeds.NotePage)
}

type ImageProcessor struct {
}

type AdminController struct {
	Queries DatabaseAccess
	Images  ImageProcessor
}

func (AdminController) AdminPage(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate session
	// TODO get build timestamp
	// TODO get config (title, description, base URL, etc.)
	// TODO get current year
	// TODO get recent images
}

func (AdminController) NewNote(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate session
	// TODO insert note into DB
}

func (AdminController) UploadImage(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate session
	// TODO process uploaded image
	// TODO insert image into DB
}

func (AdminController) DownloadImage(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate session
	// TODO start downloading image
	// TODO process downloading image
	// TODO insert image into DB
}

type AssetController struct {
	Images ImageProcessor
}

func (AssetController) HandleAssets(mux *http.ServeMux) {
	// TODO serve feed and thumbnail images
	// TODO serve static assets
}

type AuthController struct {
	Queries DatabaseAccess
}

func (AuthController) RegisterPage(w http.ResponseWriter, r *http.Request) {
	// TODO get build timestamp
	// TODO get config (title, description, base URL, etc.)
	// TODO get current year
	// TODO check for existing passkey
	// TODO ensure session isn't authenticated
}

func (AuthController) RegisterStart(w http.ResponseWriter, r *http.Request) {
	// TODO get passkey IDs
}

func (AuthController) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	// TODO insert passkey into DB
}

func (AuthController) LoginPage(w http.ResponseWriter, r *http.Request) {
	// TODO ensure session isn't authenticated
	// TODO get build timestamp
	// TODO get config (title, description, base URL, etc.)
	// TODO get current year
}

func (AuthController) LoginStart(w http.ResponseWriter, r *http.Request) {
	// TODO ensure session isn't authenticated
	// TODO insert challenge into DB
}

func (AuthController) LoginFinish(w http.ResponseWriter, r *http.Request) {
	// TODO ensure session isn't authenticated
	// TODO delete challenge from DB
	// TODO insert session into DB
}

type FeedController struct {
	Queries DatabaseAccess
}

func (FeedController) HomePage(w http.ResponseWriter, r *http.Request) {
	// TODO get build timestamp
	// TODO get config (title, description, base URL, etc.)
	// TODO get weeks for nav
	// TODO get current year
	// TODO get recent notes
}

func (FeedController) NotePage(w http.ResponseWriter, r *http.Request) {
	// TODO get build timestamp
	// TODO get config (title, description, base URL, etc.)
	// TODO get weeks for nav
	// TODO get current year
	// TODO get note
}

func (FeedController) WeekPage(w http.ResponseWriter, r *http.Request) {
	// TODO get build timestamp
	// TODO get config (title, description, base URL, etc.)
	// TODO get weeks for nav
	// TODO get current year
	// TODO get notes for week
}

func (FeedController) AtomFeed(w http.ResponseWriter, r *http.Request) {
	// TODO get config (title, description, base URL, etc.)
	// TODO get recent notes
}
