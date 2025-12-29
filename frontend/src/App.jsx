import React, { useState, useEffect } from 'react';

const API_BASE = "http://localhost:8080";
const USER_ID = "120338e1-bd44-4ee3-9258-01f57c2c5a5c"; // Для примера используем фиксированный ID

function App() {
  const [balance, setBalance] = useState(0);
  const [billId, setBillId] = useState(null);
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(false);

  // 1. Инициализация: получаем или создаем счет
  useEffect(() => {
    fetch(`${API_BASE}/payments/create/${USER_ID}`, { method: 'POST' })
      .then(res => res.json())
      .then(data => {
        setBillId(data.bill_id);
        setBalance(data.balance);
      });
    refreshOrders();
  }, []);

  const refreshOrders = () => {
    fetch(`${API_BASE}/orders/orders/${USER_ID}`)
      .then(res => res.json())
      .then(data => setOrders(Array.isArray(data) ? data : []));
  };

  // 2. Сценарий: Пополнение баланса (кнопка "+")
  const topUp = async () => {
    const amount = prompt("Введите сумму пополнения:", "100");
    if (!amount || !billId) return;

    await fetch(`${API_BASE}/payments/add/${billId}?user_id=${USER_ID}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount: parseFloat(amount) })
    });
    
    // Обновляем баланс
    const res = await fetch(`${API_BASE}/payments/balance/${billId}?user_id=${USER_ID}`);
    const data = await res.json();
    setBalance(data.balance);
  };

  // 3. Сценарий: Создание заказа и автооплата (Transactional Outbox)
  const createOrder = async () => {
    setLoading(true);
    const res = await fetch(`${API_BASE}/orders/create/${USER_ID}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount: 50.0, description: "Тестовый заказ" })
    });
    const newOrder = await res.json();
    
    // Согласно схеме, процесс асинхронный
    // Запускаем опрос статуса (polling)
    const timer = setInterval(async () => {
      const statusRes = await fetch(`${API_BASE}/orders/status/${newOrder.id}`);
      const updatedOrder = await statusRes.json();
      
      if (updatedOrder.status !== 'new') {
        clearInterval(timer);
        setLoading(false);
        refreshOrders();
        // Также обновляем баланс, так как деньги могли списаться
        topUp(); 
      }
    }, 2000);
  };

  return (
    <div style={{ padding: '20px', fontFamily: 'sans-serif' }}>
      <h1>E-commerce Dashboard</h1>
      
      <div style={{ border: '1px solid #ccc', padding: '15px', borderRadius: '8px', marginBottom: '20px' }}>
        <h3>Мой счет (ID: {billId || 'создается...'})</h3>
        <p style={{ fontSize: '24px' }}>
          Баланс: <strong>{balance} руб.</strong>
          <button onClick={topUp} style={{ marginLeft: '15px', cursor: 'pointer' }}> ➕ Пополнить</button>
        </p>
      </div>

      <button 
        onClick={createOrder} 
        disabled={loading}
        style={{ padding: '10px 20px', backgroundColor: '#007bff', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
      >
        {loading ? 'Обработка оплаты...' : 'Создать заказ (50 руб.)'}
      </button>

      <h3>История заказов</h3>
      <ul>
        {orders.map(order => (
          <li key={order.id}>
            Заказ #{order.id}: {order.amount} руб. — 
            <span style={{ fontWeight: 'bold', color: order.status === 'finished' ? 'green' : (order.status === 'canceled' ? 'red' : 'orange') }}>
               {order.status}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default App;