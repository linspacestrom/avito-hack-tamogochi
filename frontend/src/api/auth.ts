import type {
    AuthResponse,
    RefreshResponse,
    User,
} from "../types/auth";

const API_URL = "/api";

export async function login(
    email: string,
    password: string
): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/auth/login`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            email,
            password,
        }),
    });

    if (!response.ok) {
        throw new Error("Ошибка авторизации");
    }

    return response.json();
}

export async function register(
    email: string,
    password: string,
    displayName: string
): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/auth/register`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            email,
            password,
            displayName,
        }),
    });

    if (!response.ok) {
        throw new Error("Ошибка регистрации");
    }

    return response.json();
}

export async function refresh(
    refreshToken: string
): Promise<RefreshResponse> {
    const response = await fetch(`${API_URL}/auth/refresh`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            refreshToken,
        }),
    });

    if (!response.ok) {
        throw new Error("Сессия истекла");
    }

    return response.json();
}

export async function me(
    accessToken: string
): Promise<User> {
    const response = await fetch(`${API_URL}/auth/me`, {
        method: "GET",
        headers: {
            Authorization: `Bearer ${accessToken}`,
        },
    });

    if (!response.ok) {
        throw new Error("Не удалось получить пользователя");
    }

    return response.json();
}