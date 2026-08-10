import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

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
        description: "Накорми питомца 3 раза",
        progress: 2,
        target: 3,

        reward: {
            icon: "/coin.png",
            value: "+50",
            name: "монет",
        },
    },

    {
        title: "Поиграть с питомцем",
        description: "Сыграй с питомцем 3 раза",
        progress: 3,
        target: 3,

        reward: {
            icon: "/energy.png",
            value: "+20",
            name: "энергии",
        },
    },

    {
        title: "Провести день вместе",
        description: "Заходи в игру каждый день",
        progress: 0,
        target: 1,

        reward: {
            icon: "/coin.png",
            value: "+100",
            name: "монет",
        },
    },
];



const mockDailyReward = [
    {
        icon: "/coin.png",
        value: "+100",
        name: "монет",
    },

    {
        icon: "/energy.png",
        value: "+20",
        name: "энергии",
    },

    {
        icon: "/hp.png",
        value: "+50",
        name: "опыта",
    },
];



function Rewards() {


    const navigate = useNavigate();


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


            <button
                className={styles.backButton}
                onClick={() => navigate("/pet")}
            >
                ← Вернуться к питомцу
            </button>




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
                    🎁 Ежедневная награда
                </h2>



                <div className={styles.dailyCard}>


                    <div>

                        <h3>
                            День {dailyStatus?.currentDay ?? 1} из 14
                        </h3>


                        <p>

                            {
                                dailyStatus?.canClaim
                                ? "Сегодня награда доступна"
                                : "Ты уже получил награду сегодня"
                            }

                        </p>

                    </div>



                    <div className={styles.rewardList}>


                        {
                            mockDailyReward.map(item => (

                                <div
                                    className={styles.rewardItem}
                                    key={item.name}
                                >

                                    <img
                                        src={item.icon}
                                        alt={item.name}
                                    />


                                    <strong>
                                        {item.value}
                                    </strong>


                                    <span>
                                        {item.name}
                                    </span>


                                </div>

                            ))
                        }


                    </div>



                    <button

                        className={styles.claimButton}

                        disabled={
                            !dailyStatus?.canClaim
                        }

                        onClick={handleClaim}

                    >

                        {
                            dailyStatus?.canClaim
                            ? "Забрать награду"
                            : "Получено ✓"
                        }


                    </button>



                </div>


            </section>







            <section>


                <h2>
                    📋 Ежедневные задания
                </h2>



                <div className={styles.tasks}>


                    {
                        tasks.map(task => {


                            const completed =
                                task.progress >= task.target;



                            return (

                                <div
                                    className={styles.taskCard}
                                    key={task.title}
                                >


                                    <h3>
                                        {task.title}
                                    </h3>


                                    <p>
                                        {task.description}
                                    </p>



                                    <div className={styles.progress}>

                                        <div

                                            style={{
                                                width:
                                                `${(task.progress / task.target) * 100}%`
                                            }}

                                        />


                                    </div>



                                    <span>
                                        {task.progress}/{task.target}
                                    </span>




                                    <div className={styles.taskReward}>


                                        <img
                                            src={task.reward.icon}
                                            alt=""
                                        />


                                        <span>
                                            {task.reward.value} {task.reward.name}
                                        </span>


                                    </div>



                                    <button

                                        disabled={!completed}

                                    >

                                        {
                                            completed
                                            ? "Забрать"
                                            : "В процессе"
                                        }


                                    </button>



                                </div>

                            );


                        })
                    }


                </div>


            </section>







            <section>


                <h2>
                    🎒 Полученные награды
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
                        rewards.map(reward => (

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
                                    Получено ✓
                                </p>


                            </div>

                        ))
                    }



                </div>


            </section>



        </main>

    );

}


export default Rewards;