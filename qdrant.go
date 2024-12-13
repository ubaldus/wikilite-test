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

func qdrantConnect(addr string) (qdrant.CollectionsClient, qdrant.PointsClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	collectionsClient := qdrant.NewCollectionsClient(conn)
	pointsClient := qdrant.NewPointsClient(conn)
	return collectionsClient, pointsClient, nil
}

func qdrantCollectionExists(ctx context.Context, client qdrant.CollectionsClient, collectionName string) (bool, error) {
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

func qdrantCreateCollection(ctx context.Context, client qdrant.CollectionsClient, collectionName string, embeddingSize int) error {
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

func qdrantCheckIfHashExists(ctx context.Context, client qdrant.PointsClient, collectionName, hash string) (bool, error) {
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

func qdrantUpsertPoint(ctx context.Context, client qdrant.PointsClient, collectionName, hash string, embedding []float32) error {
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

func qdrantInit(ctx context.Context, addr string, collectionName string, embeddingSize int, hashMap map[string][]float32) (qdrant.CollectionsClient, qdrant.PointsClient, error) {
	collectionsClient, pointsClient, err := qdrantConnect(addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	exists, err := qdrantCollectionExists(ctx, collectionsClient, collectionName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if !exists {
		err = qdrantCreateCollection(ctx, collectionsClient, collectionName, embeddingSize)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create collection: %w", err)
		}
		fmt.Printf("Collection '%s' created successfully.\n", collectionName)
	} else {
		fmt.Printf("Collection '%s' already exists.\n", collectionName)
	}

	err = qdrantProcessHashMap(ctx, pointsClient, collectionName, hashMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process hash map: %w", err)
	}
	fmt.Println("Data upserted successfully")

	return collectionsClient, pointsClient, nil
}

func qdrantProcessHashMap(ctx context.Context, client qdrant.PointsClient, collectionName string, hashMap map[string][]float32) error {
	for hash, embedding := range hashMap {
		exists, err := qdrantCheckIfHashExists(ctx, client, collectionName, hash)
		if err != nil {
			return fmt.Errorf("failed to check if hash exists: %w", err)
		}
		if !exists {
			err = qdrantUpsertPoint(ctx, client, collectionName, hash, embedding)
			if err != nil {
				return fmt.Errorf("failed to upsert point: %w", err)
			}
			fmt.Printf("Upserted point with hash '%s'\n", hash)
		} else {
			fmt.Printf("Hash '%s' already exists, skipping.\n", hash)
		}
	}
	return nil
}

func qdrantSearch(ctx context.Context, client qdrant.PointsClient, collectionName string, vector []float32) ([]string, []float64, error) {
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
