package networking

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/14rcole/distbuilder/worker/builder"
	"github.com/sirupsen/logrus"
)

// Handles a request to build a portion of a container
//
// r.Body houses the diff of the most recent top layer, which will be applied
// to the base image before the new changes are applied
//
// The headers will include "from" - the base image, and "step" - the step
// to be applied
func BuildHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("converting body to []byte...")
	body := bodyToBytes(r.Body)
	r.Body.Close()
	var b builder.Builder

	logrus.Debug("converting to struct")
	err := json.Unmarshal(body, b)

	logrus.Debug("setting store and executor...")
	if err = b.SetStoreAndExecutor(); err != nil {
		fmt.Fprintf(w, `{"success": %t, "error": %q}`, false, err)
		return
	}

	logrus.Debug("pulling image...")
	parentLayer, err := b.PullImageIfNotExists()
	if err != nil {
		fmt.Fprintf(w, `{"success": %t, "error": %q}`, false, err)
		return
	}
	logrus.Debug("applying diff...")
	_, err = b.UseDiff(parentLayer)
	if err != nil {
		fmt.Fprintf(w, `{"success": %t, "error": %q}`, false, err)
		return
	}

	logrus.Debug("completing step.....")
	diff, err := b.DoStep()
	if err != nil {
		fmt.Fprintf(w, `{"success": %t, "error": %q}`, false, err)
		return
	}
	fmt.Fprintf(w, `{"success": %t, "diff": %q}`, true, diff)
}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

func bodyToBytes(body io.ReadCloser) []byte {
	defer body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(body)
	return buf.Bytes()
}
