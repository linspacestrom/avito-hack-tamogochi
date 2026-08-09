import { useEffect, useState } from "react";

import {
    getMyRewards,
    getDailyStatus,
    claimDailyReward,
} from "../../api/rewards";

import { useAuthContext } from "../../providers/AuthProvider";

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


function Rewards() {

    const {
        accessToken,
    } = useAuthContext();


    const [rewards, setRewards] = useState<any[]>([]);
    const [dailyStatus, setDailyStatus] = useState<any | null>(null);



    async function loadRewards() {

        if (!accessToken) return;


        try {

            const myRewards = await getMyRewards(
                accessToken
            );


            const status = await getDailyStatus(
                accessToken
            );


            setRewards(
                myRewards.rewards ?? []
            );


            setDailyStatus(
                status
            );


        } catch (error) {

            console.error(
                "Ошибка загрузки наград",
                error
            );

        }

    }



    useEffect(() => {

        loadRewards();

    }, [accessToken]);




    async function handleClaim() {

        if (!accessToken) return;


        try {

            await claimDailyReward(
                accessToken
            );


            await loadRewards();


        } catch (error) {

            console.error(
                "Ошибка получения награды",
                error
            );

        }

    }




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



            {
                dailyStatus && (

                    <section>

                        <h2>
                            Ежедневная награда
                        </h2>


                        <div className={styles.rewardCard}>

                            <h3>
                                День {dailyStatus.currentDay}
                            </h3>


                            <p>
                                {
                                    dailyStatus.canClaim
                                    ? "Награда доступна"
                                    : "Уже получено"
                                }
                            </p>


                            <button
                                onClick={handleClaim}
                                disabled={!dailyStatus.canClaim}
                            >

                                {
                                    dailyStatus.canClaim
                                    ? "Получить"
                                    : "Получено ✓"
                                }

                            </button>


                        </div>

                    </section>

                )
            }





            <section>

                <h2>
                    Ежедневные задания
                </h2>


                <div className={styles.tasks}>


                    {
                        tasks.map(task => (

                            <div
                                className={styles.taskCard}
                                key={task.title}
                            >

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

                        ))
                    }


                </div>

            </section>





            <section>

                <h2>
                    Полученные награды
                </h2>


                <div className={styles.rewards}>


                    {
                        rewards.length === 0 && (

                            <p>
                                Пока наград нет
                            </p>

                        )
                    }



                    {
                        rewards.map((reward) => (

                            <div
                                className={styles.rewardCard}
                                key={reward.id}
                            >

                                <img
                                    src="/coin.png"
                                    alt="reward"
                                />


                                <h3>
                                    Награда
                                </h3>


                                <p>
                                    {reward.status}
                                </p>


                                {
                                    reward.redeemedAt && (

                                        <span>
                                            Получено ✓
                                        </span>

                                    )
                                }


                            </div>

                        ))
                    }


                </div>


            </section>



        </main>
    );
}


export default Rewards;