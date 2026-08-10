import {
    createContext,
    useContext,
    useEffect,
    useState,
} from "react";

import {
    login as apiLogin,
    register as apiRegister,
    refresh as apiRefresh,
    me as apiMe,
} from "../api/auth";

import type { User } from "../types/auth";

interface AuthContextType {
    user: User | null;
    accessToken: string | null;
    refreshToken: string | null;
    isLoading: boolean;

    login: (email: string, password: string) => Promise<void>;

    register: (
        email: string,
        password: string,
        displayName: string
    ) => Promise<void>;

    logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({
    children,
}: {
    children: React.ReactNode;
}) {
    const [user, setUser] = useState<User | null>(null);
    const [accessToken, setAccessToken] = useState<string | null>(null);
    const [refreshToken, setRefreshToken] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        async function restoreSession() {
            const savedRefreshToken =
                localStorage.getItem("refreshToken");

            if (!savedRefreshToken) {
                setIsLoading(false);
                return;
            }

            try {
                const refreshResponse =
                    await apiRefresh(savedRefreshToken);

                setAccessToken(refreshResponse.accessToken);
                setRefreshToken(refreshResponse.refreshToken);

                localStorage.setItem(
                    "refreshToken",
                    refreshResponse.refreshToken
                );

                const currentUser = await apiMe(
                    refreshResponse.accessToken
                );

                setUser(currentUser);
            } catch {
                localStorage.removeItem("refreshToken");

                setUser(null);
                setAccessToken(null);
                setRefreshToken(null);
            } finally {
                setIsLoading(false);
            }
        }

        restoreSession();
    }, []);

    async function login(
        email: string,
        password: string
    ) {
        const response = await apiLogin(
            email,
            password
        );

        setUser(response.user);
        setAccessToken(response.accessToken);
        setRefreshToken(response.refreshToken);

        localStorage.setItem(
            "refreshToken",
            response.refreshToken
        );
    }

    async function register(
        email: string,
        password: string,
        displayName: string
    ) {
        const response = await apiRegister(
            email,
            password,
            displayName
        );

        setUser(response.user);
        setAccessToken(response.accessToken);
        setRefreshToken(response.refreshToken);

        localStorage.setItem(
            "refreshToken",
            response.refreshToken
        );
    }

    function logout() {
        setUser(null);
        setAccessToken(null);
        setRefreshToken(null);

        localStorage.removeItem("refreshToken");
    }

    return (
        <AuthContext.Provider
            value={{
                user,
                accessToken,
                refreshToken,
                isLoading,
                login,
                register,
                logout,
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export function useAuthContext() {
    const context = useContext(AuthContext);

    if (!context) {
        throw new Error(
            "useAuthContext must be used inside AuthProvider"
        );
    }

    return context;
}