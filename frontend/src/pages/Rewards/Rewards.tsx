import styles from "./Rewards.module.css";

const tasks = [
    {
        title: "Покормить питомца",
        reward: "+50 🪙",
        progress: 2,
        target: 3,
    },
    {
        title: "Поиграть с питомцем",
        reward: "+20 🪙",
        progress: 3,
        target: 3,
    },
    {
        title: "Провести день вместе",
        reward: "+100 🪙",
        progress: 0,
        target: 1,
    },
];


const rewards = [
    {
        title: "Первая награда",
        image: "/coin.png",
        description: "100 монет",
        received: true,
    },
    {
        title: "Энергия",
        image: "/energy.png",
        description: "+50 энергии",
        received: false,
    },
    {
        title: "Настроение",
        image: "/mood.png",
        description: "Бонус настроения",
        received: false,
    },
];


function Rewards() {
    return (
        <main className={styles.page}>

            <div className={styles.header}>
                <h1>
                    🏆 Награды
                </h1>

                <p>
                    Выполняй задания и развивай своего питомца
                </p>
            </div>


            <section>

                <h2>
                    Ежедневные задания
                </h2>


                <div className={styles.tasks}>

                    {tasks.map(task => (

                        <div className={styles.taskCard}>

                            <h3>
                                {task.title}
                            </h3>

                            <span>
                                Награда {task.reward}
                            </span>


                            <div className={styles.progress}>
                                <div
                                    style={{
                                        width:
                                        `${(task.progress / task.target) * 100}%`
                                    }}
                                />
                            </div>


                            <small>
                                {task.progress}/{task.target}
                            </small>


                        </div>

                    ))}

                </div>

            </section>



            <section>

                <h2>
                    Награды
                </h2>


                <div className={styles.rewards}>

                    {rewards.map(reward => (

                        <div className={styles.rewardCard}>

                            <img
                                src={reward.image}
                            />


                            <h3>
                                {reward.title}
                            </h3>


                            <p>
                                {reward.description}
                            </p>


                            <button
                                disabled={reward.received}
                            >
                                {
                                    reward.received
                                    ? "Получено ✓"
                                    : "Получить"
                                }
                            </button>


                        </div>

                    ))}

                </div>

            </section>


        </main>
    );
}


export default Rewards;