import type { Pet } from "../../../types/pet";

import styles from "./PetScene.module.css";


interface Props {
    pet: Pet;
}


function PetScene({ pet }: Props) {
    return (
        <section className={styles.scene}>

            <img
                className={styles.room}
                src="/room.png"
                alt="Комната питомца"
            />

            <img
                className={styles.cat}
                src="/cat.png"
                alt={pet.name}
            />

        </section>
    );
}


export default PetScene;