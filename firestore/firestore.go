package firestore

import (
	"context"
	"fmt"
	"regexp"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrExists   = fmt.Errorf("[FireStore] Path exists")
	ErrInvalid  = fmt.Errorf("[FireStore] Invalid Path! (collections should have odd path elements and docs should have even)")
	ErrNotFound = fmt.Errorf("[FireStore] Path Not Found")
)

type Document struct {
	Path string
	Data map[string]any
}

type FireStore struct {
	client      *firestore.Client
	project     string
	database    string
	namespace   string
	prefixRegex *regexp.Regexp
}

type Option func(*FireStore)

func WithDatabase(database string) Option {
	return func(f *FireStore) {
		f.database = database
	}
}
func WithNamespace(namespace string) Option {
	return func(f *FireStore) {
		f.namespace = namespace
	}
}

func New(ctx context.Context, project string, options ...Option) (*FireStore, error) {
	var err error
	f := &FireStore{
		project: project,
	}
	for _, option := range options {
		option(f)
	}
	if f.namespace != "" {
		f.prefixRegex = regexp.MustCompile(`^.*/namespaces/` + f.namespace + "/")
	} else {
		f.prefixRegex = regexp.MustCompile(`^.*/documents/`)
	}
	if f.database != "" {
		if f.client, err = firestore.NewClientWithDatabase(ctx, f.project, f.database); err != nil {
			return nil, err
		}
		return f, err
	}
	if f.client, err = firestore.NewClient(ctx, f.project); err != nil {
		return nil, err
	}
	return f, err
}

func (f *FireStore) prefix(path string) string {
	if f.namespace == "" {
		return path
	}
	return fmt.Sprintf("namespaces/%s/%s", f.namespace, path)
}

func (f *FireStore) strip(path string) string {
	return f.prefixRegex.ReplaceAllString(path, "")
}

func (f *FireStore) Create(ctx context.Context, path string, data map[string]any) error {
	var err error
	path = f.prefix(path)
	doc := f.client.Doc(path)
	if doc == nil {
		return ErrInvalid
	}
	if _, err = doc.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return ErrExists
		}
		return err
	}
	return nil
}

func (f *FireStore) Set(ctx context.Context, path string, data map[string]any) error {
	var err error
	path = f.prefix(path)
	doc := f.client.Doc(path)
	if doc == nil {
		return ErrInvalid
	}
	if _, err = doc.Set(ctx, data); err != nil {
		return err
	}
	return nil
}

func (f *FireStore) Delete(ctx context.Context, path string) error {
	var err error
	path = f.prefix(path)
	doc := f.client.Doc(path)
	if doc == nil {
		return ErrInvalid
	}
	if _, err = doc.Delete(ctx); err != nil {
		return err
	}
	return nil
}

func (f *FireStore) Get(ctx context.Context, path string) (map[string]any, error) {
	var err error
	path = f.prefix(path)
	doc := f.client.Doc(path)
	if doc == nil {
		return nil, ErrInvalid
	}
	var ds *firestore.DocumentSnapshot
	if ds, err = doc.Get(ctx); err != nil {
		return nil, err
	}
	if ds == nil {
		return nil, ErrNotFound
	}
	return ds.Data(), nil
}

type Condition struct {
	Path  string
	Op    string
	Value any
}

func (f *FireStore) List(ctx context.Context, path string, condition *Condition) ([]*Document, error) {
	var err error
	path = f.prefix(path)
	col := f.client.Collection(path)
	if col == nil {
		return nil, ErrInvalid
	}
	var di *firestore.DocumentIterator
	if condition != nil {
		di = col.Where(condition.Path, condition.Op, condition.Value).Documents(ctx)
	} else {
		di = col.Documents(ctx)
	}

	var docs []*Document
	var dss []*firestore.DocumentSnapshot
	if dss, err = di.GetAll(); err != nil {
		return nil, err
	}
	for _, ds := range dss {
		docs = append(docs, &Document{
			Path: f.strip(ds.Ref.Path),
			Data: ds.Data(),
		})
	}
	return docs, nil
}
