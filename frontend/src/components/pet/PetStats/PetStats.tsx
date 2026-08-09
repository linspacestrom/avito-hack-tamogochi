import type { Pet } from "../../../types/pet";

import styles from "./PetStats.module.css";


interface Props {
    pet: Pet;
}


function PetStats({ pet }: Props) {
    return (
        <aside className={styles.stats}>

            <div className={styles.level}>

                <h2>УРОВЕНЬ</h2>

                <div className={styles.levelRow}>

                    <strong>
                        {pet.level}
                    </strong>

                    <div className={styles.progress}>
                        <div
                            className={styles.progressFill}
                            style={{
                                width: `${(pet.experience / 2000) * 100}%`
                            }}
                        />
                    </div>

                </div>


                <p>
                    {pet.experience} / 2000
                </p>

            </div>



            <div className={styles.stat}>

                <div className={styles.statHeader}>

                    <img
                        src="/hunger.png"
                        alt="Сытость"
                    />

                    <span>
                        СЫТОСТЬ
                    </span>

                    <b>
                        {pet.health}/100
                    </b>

                </div>


                <div className={styles.bar}>

                    <div
                        className={styles.hungerFill}
                        style={{
                            width: `${pet.health}%`
                        }}
                    />

                </div>

            </div>



            <div className={styles.stat}>

                <div className={styles.statHeader}>

                    <img
                        src="/mood.png"
                        alt="Настроение"
                    />

                    <span>
                        НАСТРОЕНИЕ
                    </span>

                    <b>
                        {pet.happiness}/100
                    </b>

                </div>


                <div className={styles.bar}>

                    <div
                        className={styles.moodFill}
                        style={{
                            width: `${pet.happiness}%`
                        }}
                    />

                </div>

            </div>


        </aside>
    );
}


export default PetStats;