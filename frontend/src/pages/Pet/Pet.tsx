import PetScene from "../../components/pet/PetScene";
import PetStats from "../../components/pet/PetStats";
import PetActions from "../../components/pet/PetActions";
import PetHeader from "../../components/pet/PetHeader";

import styles from "./Pet.module.css";

const mockPet = {
    id: "1",
    name: "Мур",
    species: "cat",

    level: 14,
    experience: 1250,

    health: 100,
    energy: 80,
    happiness: 85,

    streakDays: 5,
};

function Pet() {

    return (
        <main>
            <PetHeader />

            <div className={styles.layout}>
                <PetStats pet={mockPet} />

                <PetScene pet={mockPet} />

                <PetActions />
            </div>
        </main>
    );
}

export default Pet;