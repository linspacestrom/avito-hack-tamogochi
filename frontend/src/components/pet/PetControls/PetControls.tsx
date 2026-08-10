import type { Dispatch, SetStateAction } from "react";
import { useNavigate } from "react-router-dom";

import type { Pet } from "../../../types/pet";

import styles from "./PetControls.module.css";


interface Props {

    pet: Pet;

    setPet: Dispatch<SetStateAction<Pet>>;

    setActionText: (text: string) => void;

}




function PetControls({
    pet,
    setPet,
    setActionText,
}: Props) {


    const navigate = useNavigate();





    const showAction = (text: string) => {


        setActionText(text);



        setTimeout(() => {

            setActionText("");

        }, 2000);


    };








    const handleAction = (title: string) => {



        switch (title) {




            case "Покормить":


                if (pet.hunger >= 100) {


                    showAction(
                        "🍗 Мур уже сыт!"
                    );


                    return;

                }





                setPet(prev => ({


                    ...prev,


                    hunger: Math.min(

                        prev.hunger + 20,

                        100

                    ),



                    happiness: Math.min(

                        prev.happiness + 5,

                        100

                    ),



                    experience:

                        prev.experience + 10,


                }));





                showAction(

                    "🍗 Мур покушал! +10 XP"

                );



                break;







            case "Играть":


                if (pet.energy < 20) {

                    showAction(
                        "😴 Мур устал, дай ему отдохнуть!"
                    );

                    return;

                }




                setPet(prev => ({


                    ...prev,


                    energy:
                        prev.energy - 20,



                    happiness: Math.min(
                        prev.happiness + 10,
                        100
                    ),



                    experience:
                        prev.experience + 20,


                }));





                showAction(
                    "🎾 Мур играет! +20 XP"
                );



                break;








            case "Награды":


                navigate("/rewards");


                break;








            case "Статистика":


                console.log(
                    "Открыть статистику"
                );


                break;




            default:

                break;


        }


    };









    const actions = [


        {
            icon: "/hunger.png",
            title: "Покормить",
        },


        {
            icon: "/mood.png",
            title: "Играть",
        },


        {
            icon: "/coin.png",
            title: "Награды",
        },


        {
            icon: "/energy.png",
            title: "Статистика",
        },


    ];








    return (


        <div className={styles.controls}>


            {
                actions.map(action => (


                    <button

                        key={action.title}

                        className={styles.button}

                        onClick={() =>
                            handleAction(
                                action.title
                            )
                        }

                    >


                        <img

                            src={action.icon}

                            alt={action.title}

                        />



                        <span>

                            {action.title}

                        </span>


                    </button>


                ))
            }



        </div>


    );


}



export default PetControls;