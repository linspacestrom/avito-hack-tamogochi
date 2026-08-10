package event

// AggregateType identifies a stream of domain events.
type AggregateType string

const (
	AggregatePet         AggregateType = "PET"
	AggregateReward      AggregateType = "REWARD"
	AggregateGameSession AggregateType = "GAME_SESSION"
)

// EventType identifies a versioned domain event payload.
type EventType string

const (
	PetCreated           EventType = "PET_CREATED"
	CareActionCompleted  EventType = "CARE_ACTION_COMPLETED"
	ExperienceGranted    EventType = "EXPERIENCE_GRANTED"
	CoinsGranted         EventType = "COINS_GRANTED"
	LevelReached         EventType = "LEVEL_REACHED"
	GameSessionCompleted EventType = "GAME_SESSION_COMPLETED"
	RewardGranted        EventType = "REWARD_GRANTED"
	RewardRedeemed       EventType = "REWARD_REDEEMED"
	RewardExpired        EventType = "REWARD_EXPIRED"
)
