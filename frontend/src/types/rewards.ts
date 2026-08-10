export interface Reward {
    id: string;
    name: string;
    description: string;
    image: string;
    claimed: boolean;
}

export interface Task {
    id: string;
    title: string;
    description: string;
    progress: number;
    target: number;
    completed: boolean;
}