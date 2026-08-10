import { FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { useAuth } from "../../hooks/useAuth";
import styles from "./AuthForm.module.css";

type AuthMode = "login" | "register";

interface AuthFormProps {
    mode: AuthMode;
}

function AuthForm({ mode }: AuthFormProps) {
    const isLogin = mode === "login";

    const navigate = useNavigate();
    const { login, register } = useAuth();

    const [displayName, setDisplayName] = useState("");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");

    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

    async function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();

        setError("");
        setLoading(true);

        try {
            if (isLogin) {
                await login(email, password);
            } else {
                await register(email, password, displayName);
            }

            navigate("/pet");
        } catch {
            setError(
                isLogin
                    ? "Не удалось войти"
                    : "Не удалось зарегистрироваться"
            );
        } finally {
            setLoading(false);
        }
    }

    return (
        <div className={styles.page}>
            <form
                className={styles.form}
                onSubmit={handleSubmit}
            >
                <h1>
                    {isLogin ? "Вход" : "Регистрация"}
                </h1>

                {!isLogin && (
                    <input
                        placeholder="Имя"
                        type="text"
                        value={displayName}
                        onChange={(event) =>
                            setDisplayName(event.target.value)
                        }
                        required
                    />
                )}

                <input
                    placeholder="Телефон или почта"
                    type="email"
                    value={email}
                    onChange={(event) =>
                        setEmail(event.target.value)
                    }
                    required
                />

                <input
                    placeholder="Пароль"
                    type="password"
                    value={password}
                    onChange={(event) =>
                        setPassword(event.target.value)
                    }
                    required
                />

                {error && <p>{error}</p>}

                <button
                    type="submit"
                    disabled={loading}
                >
                    {loading
                        ? "Загрузка..."
                        : isLogin
                            ? "Войти"
                            : "Зарегистрироваться"}
                </button>

                <p className={styles.switch}>
                    {isLogin ? (
                        <>
                            Нет аккаунта?{" "}
                            <Link to="/register">
                                Зарегистрироваться
                            </Link>
                        </>
                    ) : (
                        <>
                            Уже есть аккаунт?{" "}
                            <Link to="/login">
                                Войти
                            </Link>
                        </>
                    )}
                </p>
            </form>
        </div>
    );
}

export default AuthForm;