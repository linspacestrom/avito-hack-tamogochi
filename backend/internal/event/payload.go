package event

import "github.com/google/uuid"

type PetCreatedPayload struct {
	Name    string `json:"name"`
	Species string `json:"species"`
}

// Delta fields are signed changes produced by one care action.
type CareActionCompletedPayload struct {
	Action       string `json:"action"`
	SatietyDelta int    `json:"satietyDelta"`
	EnergyDelta  int    `json:"energyDelta"`
	MoodDelta    int    `json:"moodDelta"`
}

type GrantPayload struct {
	Amount int    `json:"amount"`
	Source string `json:"source"`
	// SourceID optionally links the grant to the task, game, or reward that produced it.
	SourceID *uuid.UUID `json:"sourceId,omitempty"`
}

type LevelReachedPayload struct {
	Level int `json:"level"`
}

type GameSessionCompletedPayload struct {
	SessionID uuid.UUID `json:"sessionId"`
	Score     int       `json:"score"`
}

type RewardPayload struct {
	UserRewardID       uuid.UUID `json:"userRewardId"`
	RewardDefinitionID uuid.UUID `json:"rewardDefinitionId"`
	// SourceEventID optionally identifies the event that caused the reward transition.
	SourceEventID *uuid.UUID `json:"sourceEventId,omitempty"`
}

// PendingEvent keeps construction behind typed helpers so callers cannot pair
// an event type with an unrelated payload shape.
type PendingEvent struct {
	eventType EventType
	payload   any
}

func NewPetCreated(payload PetCreatedPayload) PendingEvent {
	return PendingEvent{eventType: PetCreated, payload: payload}
}

func NewCareActionCompleted(payload CareActionCompletedPayload) PendingEvent {
	return PendingEvent{eventType: CareActionCompleted, payload: payload}
}

func NewExperienceGranted(payload GrantPayload) PendingEvent {
	return PendingEvent{eventType: ExperienceGranted, payload: payload}
}

func NewCoinsGranted(payload GrantPayload) PendingEvent {
	return PendingEvent{eventType: CoinsGranted, payload: payload}
}

func NewLevelReached(payload LevelReachedPayload) PendingEvent {
	return PendingEvent{eventType: LevelReached, payload: payload}
}

func NewGameSessionCompleted(payload GameSessionCompletedPayload) PendingEvent {
	return PendingEvent{eventType: GameSessionCompleted, payload: payload}
}

func NewRewardGranted(payload RewardPayload) PendingEvent {
	return PendingEvent{eventType: RewardGranted, payload: payload}
}

func NewRewardRedeemed(payload RewardPayload) PendingEvent {
	return PendingEvent{eventType: RewardRedeemed, payload: payload}
}

func NewRewardExpired(payload RewardPayload) PendingEvent {
	return PendingEvent{eventType: RewardExpired, payload: payload}
}

func (e PendingEvent) Type() EventType {
	return e.eventType
}
