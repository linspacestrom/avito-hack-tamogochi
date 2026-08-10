import styles from "./PetControls.module.css";


function PetControls() {

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