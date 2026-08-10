import type { Pet } from "../../../types/pet";

import styles from "./PetStats.module.css";

interface Props {
    pet: Pet;
}

function PetStats({ pet }: Props) {
    return (
        <aside className={styles.stats}>


            <div className={styles.level}>


                <div className={styles.levelNumber}>
                    {pet.level} уровень
                </div>


                <div className={styles.progress}>

                    <div
                        className={styles.progressFill}
                        style={{
                            width: `${(pet.experience / 2000) * 100}%`
                        }}
                    />

                </div>


                <div className={styles.exp}>


                    <span>
                        {pet.experience} / 2000 
                    </span>

                    <img
                        src="/hp.png"
                        alt="Опыт"
                    />

                </div>


            </div>



            {/* СЫТОСТЬ */}

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
                        {pet.hunger}/100
                    </b>

                </div>


                <div className={styles.bar}>

                    <div
                        className={styles.hungerFill}
                        style={{
                            width: `${pet.hunger}%`
                        }}
                    />

                </div>

            </div>



            {/* ЭНЕРГИЯ */}

            <div className={styles.stat}>

                <div className={styles.statHeader}>

                    <img
                        src="/energy.png"
                        alt="Энергия"
                    />

                    <span>
                        ЭНЕРГИЯ
                    </span>

                    <b>
                        {pet.energy}/100
                    </b>

                </div>


                <div className={styles.bar}>

                    <div
                        className={styles.energyFill}
                        style={{
                            width: `${pet.energy}%`
                        }}
                    />

                </div>

            </div>




            {/* НАСТРОЕНИЕ */}

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