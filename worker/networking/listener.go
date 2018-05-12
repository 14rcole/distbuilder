package networking

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func Listen() {
	http.HandleFunc("/", BuildHandler)
	http.HandleFunc("healthz", healthcheckHandler)
	logrus.Fatal(http.ListenAndServe(":8080", nil))
}
