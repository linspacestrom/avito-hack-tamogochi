import { useState } from "react";

import PetScene from "../../components/pet/PetScene";
import PetStats from "../../components/pet/PetStats";
import PetActions from "../../components/pet/PetActions";
import PetHeader from "../../components/pet/PetHeader";
import PetControls from "../../components/pet/PetControls";

import styles from "./Pet.module.css";


const mockPet = {
    id: "1",
    name: "Мур",
    species: "cat",

    level: 14,
    experience: 1250,

    health: 100,
    hunger: 100,
    energy: 80,
    happiness: 85,

    streakDays: 5,
};


function Pet() {

    const [selectedPet, setSelectedPet] = useState("/cat.png");


    return (
        <main>

            <PetHeader />


            <div className={styles.layout}>


                <PetStats pet={mockPet} />



                <div className={styles.sceneWrapper}>

                    <PetScene
                        pet={mockPet}
                        petImage={selectedPet}
                    />


                    <PetControls />

                </div>



                <PetActions
                    selectedPet={selectedPet}
                    onSelectPet={(pet) => setSelectedPet(pet.image)}
                />


            </div>


        </main>
    );
}


export default Pet;