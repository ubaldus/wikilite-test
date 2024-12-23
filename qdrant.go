// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantHandler struct {
	CollectionsClient qdrant.CollectionsClient
	PointsClient      qdrant.PointsClient
	Collection        string
}

func qdrantInit(host string, port int, collectionName string, embeddingSize int) (*QdrantHandler, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	collectionsClient := qdrant.NewCollectionsClient(conn)
	pointsClient := qdrant.NewPointsClient(conn)

	handler := &QdrantHandler{
		CollectionsClient: collectionsClient,
		PointsClient:      pointsClient,
		Collection:        collectionName,
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
		log.Printf("Qdrant collection '%s' created successfully.\n", collectionName)
	}

	return handler, nil
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

func qdrantUpsertPoints(client qdrant.PointsClient, collectionName string, hashEmbeddings map[string][]float32) error {
	ctx := context.Background()
	true_val := true
	points := make([]*qdrant.PointStruct, 0, len(hashEmbeddings))

	for hash, embedding := range hashEmbeddings {
		points = append(points, &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: hash,
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: embedding,
					},
				},
			},
		})
	}

	_, err := client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Wait:           &true_val,
		Points:         points,
	})
	return err
}

func qdrantSearch(client qdrant.PointsClient, collectionName string, vector []float32, limit int) ([]string, []float64, error) {
	ctx := context.Background()
	resp, err := client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Limit:          uint64(limit),
		Vector:         vector,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: false,
			},
		},
		WithVectors: &qdrant.WithVectorsSelector{
			SelectorOptions: &qdrant.WithVectorsSelector_Enable{
				Enable: false,
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search point: %w", err)
	}

	hashes := make([]string, 0, len(resp.GetResult()))
	scores := make([]float64, 0, len(resp.GetResult()))

	for _, point := range resp.GetResult() {
		if id, ok := point.Id.PointIdOptions.(*qdrant.PointId_Uuid); ok {
			hashes = append(hashes, id.Uuid)
			scores = append(scores, float64(point.Score))

		} else {
			log.Printf("Qdrant, unexpected type for point ID: %T\n", point.Id.PointIdOptions)
			continue
		}
	}
	return hashes, scores, nil
}

func qdrantHashExists(client qdrant.PointsClient, collectionName string, hash string) (bool, error) {
	ctx := context.Background()

	resp, err := client.Get(ctx, &qdrant.GetPoints{
		CollectionName: collectionName,
		Ids: []*qdrant.PointId{
			{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: hash,
				},
			},
		},
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: false,
			},
		},
		WithVectors: &qdrant.WithVectorsSelector{
			SelectorOptions: &qdrant.WithVectorsSelector_Enable{
				Enable: false,
			},
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to get point with uuid %s: %w", hash, err)
	}

	if len(resp.GetResult()) > 0 {
		return true, nil
	}

	return false, nil
}
