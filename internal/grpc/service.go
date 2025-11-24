package grpc

import (
	"context"
	"log"
	"mangahub/api"
	"mangahub/internal/manga"
	"mangahub/internal/progress"
	"mangahub/pkg/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MangaServiceServer implements the gRPC MangaService
type MangaServiceServer struct {
	api.UnimplementedMangaServiceServer
	MangaRepo    *manga.MangaRepository
	ProgressRepo *progress.ProgressRepository
}

// GetManga retrieves a manga by ID
func (s *MangaServiceServer) GetManga(ctx context.Context, req *api.GetMangaRequest) (*api.MangaResponse, error) {
	if req.MangaId == "" {
		return nil, status.Error(codes.InvalidArgument, "manga_id is required")
	}

	m, err := s.MangaRepo.GetMangaByID(req.MangaId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "manga not found")
	}

	return &api.MangaResponse{
		Manga: &api.Manga{
			Id:           m.ID,
			Title:        m.Title,
			Author:       m.Author,
			Genres:       m.Genres,
			Status:       m.Status,
			TotalChapters: int32(m.TotalChapters),
			Description:  m.Description,
			CoverUrl:     m.CoverURL,
		},
	}, nil
}

// ListManga retrieves all manga
func (s *MangaServiceServer) ListManga(ctx context.Context, req *api.ListMangaRequest) (*api.ListMangaResponse, error) {
	mangas, err := s.MangaRepo.GetAllManga()
	if err != nil {
		log.Printf("Error listing manga: %v", err)
		return nil, status.Error(codes.Internal, "failed to list manga")
	}

	grpcMangas := make([]*api.Manga, 0, len(mangas))
	for _, m := range mangas {
		grpcMangas = append(grpcMangas, &api.Manga{
			Id:           m.ID,
			Title:        m.Title,
			Author:       m.Author,
			Genres:       m.Genres,
			Status:       m.Status,
			TotalChapters: int32(m.TotalChapters),
			Description:  m.Description,
			CoverUrl:     m.CoverURL,
		})
	}

	return &api.ListMangaResponse{
		Mangas: grpcMangas,
		Total:   int32(len(grpcMangas)),
	}, nil
}

// SearchManga searches for manga by query
func (s *MangaServiceServer) SearchManga(ctx context.Context, req *api.SearchMangaRequest) (*api.ListMangaResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	mangas, err := s.MangaRepo.SearchManga(req.Query)
	if err != nil {
		log.Printf("Error searching manga: %v", err)
		return nil, status.Error(codes.Internal, "failed to search manga")
	}

	grpcMangas := make([]*api.Manga, 0, len(mangas))
	for _, m := range mangas {
		grpcMangas = append(grpcMangas, &api.Manga{
			Id:           m.ID,
			Title:        m.Title,
			Author:       m.Author,
			Genres:       m.Genres,
			Status:       m.Status,
			TotalChapters: int32(m.TotalChapters),
			Description:  m.Description,
			CoverUrl:     m.CoverURL,
		})
	}

	return &api.ListMangaResponse{
		Mangas: grpcMangas,
		Total:   int32(len(grpcMangas)),
	}, nil
}

// GetUserProgress retrieves user's reading progress
func (s *MangaServiceServer) GetUserProgress(ctx context.Context, req *api.GetUserProgressRequest) (*api.UserProgressResponse, error) {
	if req.UserId == "" || req.MangaId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and manga_id are required")
	}

	p, err := s.ProgressRepo.GetMangaProgress(req.UserId, req.MangaId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "progress not found")
	}

	return &api.UserProgressResponse{
		Progress: &api.UserProgress{
			Id:        p.ID,
			UserId:    p.UserID,
			MangaId:   p.MangaID,
			Chapter:   int32(p.Chapter),
			UpdatedAt: p.UpdatedAt,
		},
	}, nil
}

// UpdateProgress updates user's reading progress
func (s *MangaServiceServer) UpdateProgress(ctx context.Context, req *api.UpdateProgressRequest) (*api.UpdateProgressResponse, error) {
	if req.UserId == "" || req.MangaId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and manga_id are required")
	}

	if req.Chapter < 0 {
		return nil, status.Error(codes.InvalidArgument, "chapter must be non-negative")
	}

	progress := models.UserProgress{
		ID:      req.UserId + "_" + req.MangaId, // Simple ID generation
		UserID:  req.UserId,
		MangaID: req.MangaId,
		Chapter: int(req.Chapter),
	}

	if err := s.ProgressRepo.UpdateProgress(progress); err != nil {
		log.Printf("Error updating progress: %v", err)
		return nil, status.Error(codes.Internal, "failed to update progress")
	}

	return &api.UpdateProgressResponse{
		Success: true,
		Message: "Progress updated successfully",
		Progress: &api.UserProgress{
			Id:        progress.ID,
			UserId:    progress.UserID,
			MangaId:   progress.MangaID,
			Chapter:   int32(progress.Chapter),
			UpdatedAt: progress.UpdatedAt,
		},
	}, nil
}

