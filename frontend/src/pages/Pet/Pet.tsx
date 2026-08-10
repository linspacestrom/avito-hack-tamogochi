import { useEffect, useState } from "react";

import PetScene from "../../components/pet/PetScene";
import PetStats from "../../components/pet/PetStats";
import PetActions from "../../components/pet/PetActions";
import PetHeader from "../../components/pet/PetHeader";
import PetControls from "../../components/pet/PetControls";

import styles from "./Pet.module.css";


type PetState = {
    id: string;
    name: string;
    species: string;

    level: number;
    experience: number;

    health: number;
    hunger: number;
    energy: number;
    happiness: number;

    streakDays: number;
};



const initialPet: PetState = {

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


    const [pet, setPet] = useState<PetState>(() => {


        try {

            const savedPet =
                localStorage.getItem("pet");


            return savedPet

                ? JSON.parse(savedPet) as PetState

                : initialPet;


        } catch {

            return initialPet;

        }


    });






    const [selectedPet, setSelectedPet] = useState<string>(() => {


        return (

            localStorage.getItem("selectedPet")

            ||

            "/cat.png"

        );


    });






    const [actionText, setActionText] = useState("");







    useEffect(() => {


        localStorage.setItem(

            "pet",

            JSON.stringify(pet)

        );


    }, [pet]);








    useEffect(() => {


        localStorage.setItem(

            "selectedPet",

            selectedPet

        );


    }, [selectedPet]);









    useEffect(() => {


        const timer = setInterval(() => {


            setPet(prev => ({


                ...prev,


                energy: Math.min(
                    prev.energy + 1,
                    100
                ),



                hunger: Math.max(
                    prev.hunger - 1,
                    0
                ),



                happiness:

                    prev.hunger < 30

                        ? Math.max(
                            prev.happiness - 1,
                            0
                        )

                        : prev.happiness,


            }));


        }, 10000);




        return () => {

            clearInterval(timer);

        };


    }, []);







    return (

        <main>


            <PetHeader />



            <div className={styles.layout}>




                <PetStats

                    pet={pet}

                />






                <div className={styles.sceneWrapper}>


                    {
                        actionText && (

                            <div className={styles.actionText}>

                                {actionText}

                            </div>

                        )
                    }





                    <PetScene

                        pet={pet}

                        petImage={selectedPet}

                    />







                    <PetControls

                        pet={pet}

                        setPet={setPet}

                        setActionText={setActionText}

                    />



                </div>








                <PetActions

                    selectedPet={selectedPet}

                    onSelectPet={(pet) => {


                        setSelectedPet(

                            pet.image

                        );


                    }}

                />




            </div>



        </main>

    );


}



export default Pet;