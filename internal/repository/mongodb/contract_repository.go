package mongodb

import (
	"context"
	"errors"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type contractRepository struct {
	collection     *mongo.Collection
	directHireColl *mongo.Collection
}

func NewContractRepository(db *mongo.Database) domain.ContractRepository {
	return &contractRepository{
		collection:     db.Collection("contracts"),
		directHireColl: db.Collection("direct_hire_requests"),
	}
}

func (r *contractRepository) Create(ctx context.Context, contract *domain.Contract) error {
	if contract.ID == "" {
		contract.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, contract)
	return err
}

func (r *contractRepository) GetByID(ctx context.Context, id string) (*domain.Contract, error) {
	var contract domain.Contract
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&contract)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("contract not found")
		}
		return nil, err
	}
	return &contract, nil
}

func (r *contractRepository) GetByJobID(ctx context.Context, jobID string) (*domain.Contract, error) {
	var contract domain.Contract
	err := r.collection.FindOne(ctx, bson.M{"job_id": jobID}).Decode(&contract)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("contract not found")
		}
		return nil, err
	}
	return &contract, nil
}

func (r *contractRepository) Update(ctx context.Context, contract *domain.Contract) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": contract.ID}, contract)
	return err
}

func (r *contractRepository) ListForUser(ctx context.Context, userID, role string) ([]*domain.Contract, error) {
	query := bson.M{}
	if role == "client" {
		query["client_id"] = userID
	} else if role == "musician" {
		query["musician_id"] = userID
	} else {
		query["$or"] = []bson.M{
			{"client_id": userID},
			{"musician_id": userID},
		}
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var contracts []*domain.Contract
	for cursor.Next(ctx) {
		var c domain.Contract
		if err := cursor.Decode(&c); err != nil {
			return nil, err
		}
		contracts = append(contracts, &c)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return contracts, nil
}

func (r *contractRepository) CreateDirectHireRequest(ctx context.Context, req *domain.DirectHireRequest) error {
	if req.ID == "" {
		req.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.directHireColl.InsertOne(ctx, req)
	return err
}

func (r *contractRepository) GetDirectHireRequestByID(ctx context.Context, id string) (*domain.DirectHireRequest, error) {
	var req domain.DirectHireRequest
	err := r.directHireColl.FindOne(ctx, bson.M{"_id": id}).Decode(&req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("direct hire request not found")
		}
		return nil, err
	}
	return &req, nil
}

func (r *contractRepository) UpdateDirectHireRequest(ctx context.Context, req *domain.DirectHireRequest) error {
	_, err := r.directHireColl.ReplaceOne(ctx, bson.M{"_id": req.ID}, req)
	return err
}

func (r *contractRepository) ListDirectHireRequestsForUser(ctx context.Context, userID, role, status string) ([]*domain.DirectHireRequest, error) {
	query := bson.M{}
	if role == "client" {
		query["client_id"] = userID
	} else if role == "musician" {
		query["musician_id"] = userID
	} else {
		query["$or"] = []bson.M{
			{"client_id": userID},
			{"musician_id": userID},
		}
	}
	if status != "" {
		query["status"] = status
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := r.directHireColl.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []*domain.DirectHireRequest
	for cursor.Next(ctx) {
		var req domain.DirectHireRequest
		if err := cursor.Decode(&req); err != nil {
			return nil, err
		}
		requests = append(requests, &req)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return requests, nil
}
