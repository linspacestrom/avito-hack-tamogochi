import type { Pet } from "../../../types/pet";

import styles from "./PetScene.module.css";

interface Props {
    pet: Pet;
    petImage?: string;
}

function PetScene({ pet, petImage = "/cat.png" }: Props) {
    return (
        <section className={styles.scene}>
            <img
                className={styles.room}
                src="/room.png"
                alt="Комната питомца"
            />

            <div className={styles.catWrapper}>
                <img
                    className={styles.cat}
                    src={petImage}
                    alt={pet.name}
                />
            </div>
        </section>
    );
}

export default PetScene;