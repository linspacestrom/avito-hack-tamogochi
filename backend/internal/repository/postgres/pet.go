package postgres

type PetRepository struct {
	repo *Repository
}

func NewPetRepository(repo *Repository) *PetRepository {
	return &PetRepository{
		repo: repo,
	}
}
