export interface Pet {
    id: string;
    name: string;
    species: string;

    level: number;
    experience: number;

    health: number;
    hunger: number;
    energy: number;
    happiness: number;

    streakDays: number;
}