const API_URL = "/api";


export async function getMyRewards(
    accessToken: string
) {
    const response = await fetch(
        `${API_URL}/rewards/me`,
        {
            headers: {
                Authorization: `Bearer ${accessToken}`,
            },
        }
    );


    if (!response.ok) {
        throw new Error("Ошибка получения наград");
    }


    return response.json();
}



export async function getDailyStatus(
    accessToken: string
) {
    const response = await fetch(
        `${API_URL}/rewards/daily-status`,
        {
            headers: {
                Authorization: `Bearer ${accessToken}`,
            },
        }
    );


    if (!response.ok) {
        throw new Error("Ошибка получения статуса");
    }


    return response.json();
}



export async function claimDailyReward(
    accessToken: string
) {
    const response = await fetch(
        `${API_URL}/rewards/daily-claim`,
        {
            method: "POST",
            headers: {
                Authorization: `Bearer ${accessToken}`,
                "Content-Type": "application/json",
            },
        }
    );


    if (!response.ok) {
        throw new Error("Ошибка получения награды");
    }


    return response.json();
}