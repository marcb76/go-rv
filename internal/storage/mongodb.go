package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const mongoOperationTimeout = 5 * time.Second

// MongoStore persists URL records in a MongoDB collection.
type MongoStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

type mongoURLRecord struct {
	Key           string    `bson:"_id"`
	URL           string    `bson:"url"`
	ShortURL      string    `bson:"short_url"`
	Code          string    `bson:"code"`
	AiTags        []string  `bson:"ai_tags,omitempty"`
	AiDescription string    `bson:"ai_description,omitempty"`
	Hits          int64     `bson:"hits"`
	CreatedAt     time.Time `bson:"created_at"`
}

// NewMongoStore connects to MongoDB Atlas, verifies connectivity, and ensures
// that short codes are unique within the collection.
func NewMongoStore(ctx context.Context, server, username, password, database, collection string) (*MongoStore, error) {
	if server == "" || username == "" || password == "" || database == "" || collection == "" {
		return nil, errors.New("storage: MongoDB server, username, password, database, and collection are required")
	}

	connectionURL := url.URL{
		Scheme:   "mongodb+srv",
		Host:     server,
		Path:     "/",
		RawQuery: "authSource=admin",
		User:     url.UserPassword(username, password),
	}
	connectionURI := connectionURL.String()
	client, err := mongo.Connect(options.Client().ApplyURI(connectionURI))
	if err != nil {
		return nil, fmt.Errorf("storage: connect to MongoDB: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(connectCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("storage: ping MongoDB: %w", err)
	}

	store := &MongoStore{
		client:     client,
		collection: client.Database(database).Collection(collection),
	}
	_, err = store.collection.Indexes().CreateOne(connectCtx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("unique_short_code"),
	})
	if err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("storage: create unique short-code index: %w", err)
	}

	return store, nil
}

// Close disconnects the MongoDB client.
func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// Get retrieves a record by its normalized URL key.
func (s *MongoStore) Get(key string) (*URLRecord, bool, error) {
	_, record, exists, err := s.getOne(bson.D{{Key: "_id", Value: key}})
	return record, exists, err
}

// GetByShortURL retrieves a record by its complete short URL and returns its key.
func (s *MongoStore) GetByShortURL(shortURL string) (string, *URLRecord, bool, error) {
	return s.getOne(bson.D{{Key: "short_url", Value: shortURL}})
}

// GetByCode retrieves a record by its short code and returns its key.
func (s *MongoStore) GetByCode(code string) (string, *URLRecord, bool, error) {
	return s.getOne(bson.D{{Key: "code", Value: code}})
}

func (s *MongoStore) getOne(filter bson.D) (string, *URLRecord, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mongoOperationTimeout)
	defer cancel()

	var document mongoURLRecord
	err := s.collection.FindOne(ctx, filter).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}
	return document.Key, document.toURLRecord(), true, nil
}

// Set inserts a record. A duplicate normalized key returns false without error;
// a duplicate short code returns ErrDuplicateCode so the controller can retry.
func (s *MongoStore) Set(key string, record *URLRecord) (bool, error) {
	if record == nil {
		return false, ErrNilURLRecord
	}

	ctx, cancel := context.WithTimeout(context.Background(), mongoOperationTimeout)
	defer cancel()
	_, err := s.collection.InsertOne(ctx, newMongoURLRecord(key, cloneRecord(record)))
	if err == nil {
		return true, nil
	}
	if !mongo.IsDuplicateKeyError(err) {
		return false, err
	}

	_, _, exists, lookupErr := s.getOne(bson.D{{Key: "_id", Value: key}})
	if lookupErr != nil {
		return false, fmt.Errorf("check duplicate URL key: %w", lookupErr)
	}
	if exists {
		return false, nil
	}
	return false, ErrDuplicateCode
}

// IncrementHits atomically increments and returns the updated record.
func (s *MongoStore) IncrementHits(key string) (*URLRecord, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mongoOperationTimeout)
	defer cancel()

	var document mongoURLRecord
	err := s.collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: key}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "hits", Value: 1}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return document.toURLRecord(), true, nil
}

// All returns every URL record indexed by its normalized URL key.
func (s *MongoStore) All() (map[string]URLRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mongoOperationTimeout)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make(map[string]URLRecord)
	for cursor.Next(ctx) {
		var document mongoURLRecord
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		records[document.Key] = *document.toURLRecord()
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func newMongoURLRecord(key string, record *URLRecord) mongoURLRecord {
	return mongoURLRecord{
		Key:           key,
		URL:           record.URL,
		ShortURL:      record.ShortURL,
		Code:          record.Code,
		AiTags:        append([]string(nil), record.AiTags...),
		AiDescription: record.AiDescription,
		Hits:          record.Hits,
		CreatedAt:     record.CreatedAt,
	}
}

func (d mongoURLRecord) toURLRecord() *URLRecord {
	return &URLRecord{
		URL:           d.URL,
		ShortURL:      d.ShortURL,
		Code:          d.Code,
		AiTags:        append([]string(nil), d.AiTags...),
		AiDescription: d.AiDescription,
		Hits:          d.Hits,
		CreatedAt:     d.CreatedAt,
	}
}

var _ Store = (*MongoStore)(nil)
