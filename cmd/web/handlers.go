// Package main is the main package for this application.
package main

// Import the necessary packages.
import (
	"errors"   // Package for creating error messages.
	"fmt"      // Package for formatted I/O.
	"net/http" // Package for building HTTP servers and clients.
	"strconv"  // Package for converting strings to numeric types.

	"github.com/julienschmidt/httprouter" // Import advanced routing and validation package

	"snippetbox.adcon.dev/internal/models"    // Import the models package.
	"snippetbox.adcon.dev/internal/validator" // Import validator package
)

// snippetCreateForm represents the form that captures user input for creating a new snippet.
// It includes fields for the title, content, and expiration of the snippet, as well as a Validator
// for validating the form fields.
type snippetCreateForm struct {
	Title               string     `form:"title"`   // Title is the title of the snippet provided by the user.
	Content             string     `form:"content"` // Content is the actual code snippet provided by the user.
	Expires             int        `form:"expires"` // Expires is the duration after which the snippet expires.
	validator.Validator `form:"-"` // Validator is used to validate the form fields.
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

// home serves the root URL ("/"). It fetches the most recent snippets from the database
// and renders them on the home page. If an error occurs (for example, a database error),
// it sends a server error response.
func (app *application) home(w http.ResponseWriter, r *http.Request) {
	// Fetch the latest snippets from the database.
	// The Latest method is expected to return the most recent snippets.
	snippets, err := app.snippets.Latest()

	// If there's an error (for example, a database error), send a server error response.
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Create a new template data map and add the snippets to it.
	// This map will be passed to the template for rendering.
	data := app.newTemplateData(r)
	data.SnippetsData = snippets

	// Render the home page with the snippets.
	// The render method is expected to render the "home.html" template with the provided data.
	app.render(w, http.StatusOK, "home.html", data)
}

// snippetView serves the "/snippet/view" URL. It fetches a snippet with a given ID from the database
// and renders it on the page. If the snippet is not found or an error occurs, it sends an appropriate HTTP response.
func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {

	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	// If the ID is not a valid integer or is less than 1, respond with a 404 status.
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	// Fetch the snippet with the given ID from the database.
	snippet, err := app.snippets.Get(id)
	// If an error occurs, handle it appropriately.
	if err != nil {
		// If no snippet with the given ID was found, respond with a 404 status.
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			// For any other kind of error, respond with a 500 status.
			app.serverError(w, err)
		}
		return
	}

	// If no error occurs, create a new template data map and add the snippet to it.
	data := app.newTemplateData(r)
	data.SnippetData = snippet

	// Render the "view.html" template with the provided data.
	app.render(w, http.StatusOK, "view.html", data)
}

// snippetCreate serves the "/snippet/create" URL. It initializes a new snippetCreateForm
// with a default expiration of 365 days and renders the "create.html" template.
// This method is used to display the form for creating a new snippet.
func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	// Create a new template data map.
	data := app.newTemplateData(r)

	// Initialize a new snippetCreateForm with a default expiration of 365 days.
	data.Form = snippetCreateForm{
		Expires: 365,
	}

	// Render the "create.html" template with the provided data.
	app.render(w, http.StatusOK, "create.html", data)
}

// snippetCreatePost serves the "/snippet/create" URL for POST requests. It validates the form data
// provided by the user and, if valid, inserts a new snippet into the database. If the form data is
// not valid, it re-renders the form with error messages. If there's an error inserting the snippet
// into the database, it sends a server error response. If the snippet is inserted successfully,
// it redirects the client to the page for the new snippet.
func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {

	var form snippetCreateForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate the form values.
	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxRunes(form.Title, 100), "title", "This field cannot be more than 100 characters long")
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
	form.CheckField(validator.AllowedValue(form.Expires, 1, 7, 365), "expires", "This field must equal 1, 7 or 365")

	// If the form is not valid, re-render the form with error messages.
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "create.html", data)
		return
	}

	// Insert the new snippet into the database.
	id, err := app.snippets.Insert(form.Title, form.Content, form.Expires)
	// If there's an error (for example, a database error), send a server error response.
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")

	// If there's no error, the snippet was inserted successfully.
	// Redirect the client to the page for the new snippet.
	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {

	data := app.newTemplateData(r)
	data.Form = userSignupForm{}

	app.render(w, http.StatusOK, "signup.html", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {

	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.MaxRunes(form.Name, 255), "name", "Field is too long (255)")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinRunes(form.Password, 8), "password", "This field must be at least 8 characters long")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.html", data)
		return
	}

	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {

	data := app.newTemplateData(r)
	data.Form = userLoginForm{}

	app.render(w, http.StatusOK, "login.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {

	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form

		app.render(w, http.StatusUnprocessableEntity, "login.html", data)
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form

			app.render(w, http.StatusUnprocessableEntity, "login.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {

	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func ping(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("OK"))
}
