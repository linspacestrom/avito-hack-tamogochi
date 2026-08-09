import { useNavigate } from "react-router-dom";

import { useAuth } from "../../../hooks/useAuth";

import styles from "./PetHeader.module.css";

function PetHeader() {
    const { logout } = useAuth();
    const navigate = useNavigate();

    function handleLogout() {
        logout();
        navigate("/login");
    }

    return (
        <header className={styles.header}>

            <button
                type="button"
                className={styles.logoutButton}
                onClick={handleLogout}
            >
                Выйти
            </button>

            <img
                className={styles.logo}
                src="/avito-logo.png"
                alt="Avito"
            />

            <div className={styles.resources}>

                <div className={styles.card}>
                    <img
                        src="/coin.png"
                        alt="Coins"
                    />

                    <span>
                        12 450
                    </span>
                </div>

                <div className={styles.card}>
                    <img
                        src="/energy.png"
                        alt="Energy"
                    />

                    <span>
                        80/100
                    </span>
                </div>

            </div>

        </header>
    );
}

export default PetHeader;