package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	is "github.com/containers/image/storage"
	"github.com/containers/storage"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/fatih/structs"
	"github.com/openshift/imagebuilder"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/imagebuildah"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

// Builder manages the fields required for building an image
type Builder struct {
	Builder  *imagebuilder.Builder
	Options  imagebuildah.BuildOptions
	From     string
	Node     *parser.Node
	Diff     []byte
	store    storage.Store
	executor *imagebuildah.Executor
}

// MarshalJSON converts a Builder into a json []byte.  This must be
// done (partially) manually, as ParallelBuilder.Options includes a func()
// parameter that cannot be parsed by json.Marshal
func (b *Builder) MarshalJSON() ([]byte, error) {
	s := structs.New(b)
	buildMap := s.Map()
	optMap := buildMap["Options"].(map[string]interface{})
	delete(optMap, "Log")
	return json.Marshal(buildMap)
}

// SetStoreAndExecutor sets the container store for the node, then generates an
// executor from the store and BuildOptions
func (b *Builder) SetStoreAndExecutor() error {
	options := storage.DefaultStoreOptions
	logrus.Debugf("getting store, options: %+v", options)
	store, err := storage.GetStore(options)
	if err != nil {
		return err
	} else if store != nil {
		logrus.Debug("setting store...")
		is.Transport.SetStore(store)
	}
	b.store = store
	logrus.Debug("creating executor...")
	b.executor, err = imagebuildah.NewExecutor(store, b.Options)
	return err
}

// PullImageIfNotExists checks if an image matching the given name, id, or ref
// exists in containers/storage and pulls the image if it does not exist
func (b *Builder) PullImageIfNotExists() (string, error) {
	img, err := b.getImage(b.From)
	if err != nil {
		// Pull the image
		ref, err2 := b.pullImage(b.From)
		if err2 != nil {
			return "", errors.Wrapf(err2, "could not pull image")
		}
		img, err2 = is.Transport.GetStoreImage(b.store, ref)
		if err2 != nil {
			return "", errors.Wrapf(err2, "Could not convert image reference to storage.Image")
		}
	}
	return img.TopLayer, nil
}

// UseDiff Applies a diff and changes the fromimage of the builder to the new layer
func (b *Builder) UseDiff(parent string) (string, error) {
	diff := bytes.NewBuffer(b.Diff)
	id, err := b.ApplyDiff(parent, diff)
	if err != nil {
		return "", errors.Wrapf(err, "could not apply diff")
	}
	name := fmt.Sprintf("%s-%d", b.From, len(b.Diff))
	img, err := b.store.CreateImage("", []string{name}, id, "", nil)
	if err != nil {
		return "", errors.Wrapf(err, "could not create image from new layer")
	}
	b.From = img.ID
	return img.ID, nil
}

// ApplyDiff creates a new layer with the given diff to the layer with id parent
func (b *Builder) ApplyDiff(parent string, diff io.Reader) (string, error) {
	// create a new layer with parent as its parent layer
	layer, _, err := b.store.PutLayer("", parent, []string{}, "", true, nil, diff)
	if err != nil {
		return "", err
	}
	return layer.ID, nil
}

// DoStep completes a given step from a Dockefile and returns the result
func (b *Builder) DoStep() ([]byte, error) {
	if b.Node == nil {
		return nil, errors.New("No instruction specified")
	}
	ib := imagebuilder.NewBuilder(b.Options.Args)
	if err := b.executor.Prepare(nil, ib, b.Node, b.From); err != nil {
		return nil, errors.Wrapf(err, "could not prepare build step")
	}
	defer b.executor.Delete()
	if err := b.executor.Execute(ib, b.Node); err != nil {
		return nil, errors.Wrapf(err, "could not execute step")
	}
	if err := b.executor.Commit(nil, ib); err != nil {
		return nil, errors.Wrapf(err, "could not commit step")
	}
	image, err := b.getImage(b.From)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get image after committing")
	}
	diff, err := b.store.Diff("", image.TopLayer, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get most recent diff")
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(diff)
	return buf.Bytes(), nil
}

func (b *Builder) getImage(image string) (*storage.Image, error) {
	var img *storage.Image
	ref, err := is.Transport.ParseStoreReference(b.store, image)
	if err == nil {
		img, err = is.Transport.GetStoreImage(b.store, ref)
	}
	if err != nil {
		img2, err2 := b.store.Image(image)
		if err2 != nil {
			if ref == nil {
				return nil, errors.Wrapf(err, "error parsing reference to image %q", image)
			}
			return nil, errors.Wrapf(err, "unable to locate image %q", image)
		}
		img = img2
	}
	return img, nil
}
