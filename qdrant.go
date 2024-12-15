package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantHandler struct {
	CollectionsClient qdrant.CollectionsClient
	PointsClient      qdrant.PointsClient
	Collection        string
}

func qdrantConnect(addr, collection string) (*QdrantHandler, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	collectionsClient := qdrant.NewCollectionsClient(conn)
	pointsClient := qdrant.NewPointsClient(conn)

	return &QdrantHandler{
		CollectionsClient: collectionsClient,
		PointsClient:      pointsClient,
		Collection:        collection,
	}, nil
}

func qdrantCollectionExists(client qdrant.CollectionsClient, collectionName string) (bool, error) {
	ctx := context.Background()
	resp, err := client.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return false, err
	}

	for _, collection := range resp.GetCollections() {
		if collection.Name == collectionName {
			return true, nil
		}
	}
	return false, nil
}

func qdrantCreateCollection(client qdrant.CollectionsClient, collectionName string, embeddingSize int) error {
	ctx := context.Background()
	_, err := client.Create(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(embeddingSize),
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	return err
}

func qdrantCheckIfHashExists(client qdrant.PointsClient, collectionName, hash string) (bool, error) {
	ctx := context.Background()
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "hash",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Text{
								Text: hash,
							},
						},
					},
				},
			},
		},
	}

	resp, err := client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Limit:          1,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
		Filter: filter,
	})

	if err != nil {
		return false, fmt.Errorf("failed to search point: %w", err)
	}

	return len(resp.GetResult()) > 0, nil
}

func qdrantUpsertPoint(client qdrant.PointsClient, collectionName, hash string, embedding []float32) error {
	ctx := context.Background()
	true_val := true
	_, err := client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Wait:           &true_val,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{
					PointIdOptions: &qdrant.PointId_Num{
						Num: uint64(rand.Int()),
					},
				},
				Vectors: &qdrant.Vectors{
					VectorsOptions: &qdrant.Vectors_Vector{
						Vector: &qdrant.Vector{
							Data: embedding,
						},
					},
				},
				Payload: map[string]*qdrant.Value{
					"hash": {
						Kind: &qdrant.Value_StringValue{
							StringValue: hash,
						},
					},
				},
			},
		},
	})
	return err
}

func qdrantInit(host string, port int, collectionName string, embeddingSize int) (*QdrantHandler, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	handler, err := qdrantConnect(addr, collectionName)
	if err != nil {
		return nil, err
	}

	exists, err := qdrantCollectionExists(handler.CollectionsClient, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if !exists {
		err = qdrantCreateCollection(handler.CollectionsClient, collectionName, embeddingSize)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
		log.Printf("Collection '%s' created successfully.\n", collectionName)
	}

	return handler, nil
}

func qdrantProcessHashMap(client qdrant.PointsClient, collectionName string, hashMap map[string][]float32) error {
	for hash, embedding := range hashMap {
		exists, err := qdrantCheckIfHashExists(client, collectionName, hash)
		if err != nil {
			return fmt.Errorf("failed to check if hash exists: %w", err)
		}
		if !exists {
			err = qdrantUpsertPoint(client, collectionName, hash, embedding)
			if err != nil {
				return fmt.Errorf("failed to upsert point: %w", err)
			}
			log.Printf("Upserted point with hash '%s'\n", hash)
		}
	}
	return nil
}

func qdrantSearch(client qdrant.PointsClient, collectionName string, vector []float32) ([]string, []float64, error) {
	ctx := context.Background()
	resp, err := client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Limit:          10,
		Vector:         vector,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search point: %w", err)
	}

	hashes := make([]string, 0, len(resp.GetResult()))
	scores := make([]float64, 0, len(resp.GetResult()))

	for _, point := range resp.GetResult() {
		hashValue, ok := point.Payload["hash"]
		if !ok {
			log.Println("No hash value found in payload for a point, skipping.")
			continue
		}

		hash, ok := hashValue.Kind.(*qdrant.Value_StringValue)
		if !ok {
			log.Printf("Unexpected type for hash value: %T\n", hashValue.Kind)
			continue
		}
		hashes = append(hashes, hash.StringValue)
		scores = append(scores, float64(point.Score))
	}
	return hashes, scores, nil
}
