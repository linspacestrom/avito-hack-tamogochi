import { useState } from "react";

import styles from "./PetActions.module.css";

export interface PetOption {
    id: string;
    name: string;
    image: string;
}

const pets: PetOption[] = [
    {
        id: "cat",
        name: "Мур",
        image: "/cat.png",
    },
    {
        id: "cat2",
        name: "Барсик",
        image: "/cat2.png",
    },
    {
        id: "cat3",
        name: "Тигра",
        image: "/cat3.png",
    },
    {
        id: "cat4",
        name: "Тупак Шакур",
        image: "/cat4.png",
    },
    {
        id: "cat5",
        name: "Снежок",
        image: "/cat5.png",
    },
];

interface PetActionsProps {
    selectedPet: string;
    onSelectPet: (pet: PetOption) => void;
}

function PetActions({
    selectedPet,
    onSelectPet,
}: PetActionsProps) {
    const [open, setOpen] = useState<string | null>(null);

    const toggle = (name: string) => {
        setOpen(open === name ? null : name);
    };

    return (
        <aside className={styles.actions}>
            <div className={styles.item}>
                <button
                    className={styles.accordionButton}
                    onClick={() => toggle("pets")}
                >
                    <span>Выбор питомца</span>

                    <span>
                        {open === "pets" ? "▲" : "▼"}
                    </span>
                </button>

                {open === "pets" && (
                    <div className={styles.content}>
                        <div className={styles.petList}>
                            {pets.map((pet) => (
                                <button
                                    key={pet.id}
                                    className={`${styles.petOption} ${selectedPet === pet.image
                                            ? styles.selected
                                            : ""
                                        }`}
                                    onClick={() => onSelectPet(pet)}
                                >
                                    <img
                                        src={pet.image}
                                        alt={pet.name}
                                    />

                                    <span>{pet.name}</span>
                                </button>
                            ))}
                        </div>
                    </div>
                )}
            </div>

            <div className={styles.item}>
                <button
                    className={styles.accordionButton}
                    onClick={() => toggle("accessories")}
                >
                    <span>Аксессуары</span>

                    <span>
                        {open === "accessories" ? "▲" : "▼"}
                    </span>
                </button>

                {open === "accessories" && (
                    <div className={styles.content}>
                        Список аксессуаров будет здесь
                    </div>
                )}
            </div>
        </aside>
    );
}

export default PetActions;