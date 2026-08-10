export interface User {
    id: string;
    email: string;
    displayName: string;
    status: string;
    createdAt: string;
    updatedAt: string;
}

export interface AuthResponse {
    user: User;
    accessToken: string;
    refreshToken: string;
}

export interface RefreshResponse {
    accessToken: string;
    refreshToken: string;
}