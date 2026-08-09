import { useState } from "react";

import styles from "./PetActions.module.css";


function PetActions() {

    const [open, setOpen] = useState<string | null>(null);


    const toggle = (name: string) => {
        setOpen(
            open === name ? null : name
        );
    };


    return (
        <aside className={styles.actions}>


            <div className={styles.item}>

                <button
                    onClick={() => toggle("pets")}
                >
                    <span>
                        Выбор питомца
                    </span>

                    <span>
                        {open === "pets" ? "▲" : "▼"}
                    </span>

                </button>


                {
                    open === "pets" && (

                        <div className={styles.content}>
                            Список питомцев будет здесь
                        </div>

                    )
                }

            </div>



            <div className={styles.item}>

                <button
                    onClick={() => toggle("accessories")}
                >

                    <span>
                        Аксессуары
                    </span>

                    <span>
                        {open === "accessories" ? "▲" : "▼"}
                    </span>

                </button>


                {
                    open === "accessories" && (

                        <div className={styles.content}>
                            Список аксессуаров будет здесь
                        </div>

                    )
                }

            </div>


        </aside>
    );
}


export default PetActions;