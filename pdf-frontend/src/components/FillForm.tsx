import React, { useState } from 'react';
import axios from 'axios';

const FillForm: React.FC<{ fileName: string }> = ({ fileName }) => {
    const [name, setName] = useState('');
    const [value, setValue] = useState('');
    const [submitted, setSubmitted] = useState(false);

    const handleSubmit = async () => {
        const payload = {
            fields: [{ name, value }]
        };

        try {
            const res = await axios.post(`http://localhost:8080/fill/${fileName}`, payload);
            if (res.status === 200) {
                const finalPDF = res.data.finalPDF;
                setSubmitted(true);
                window.open(`http://localhost:8080/download/${finalPDF}`, '_blank');
            }
        } catch (err) {
            alert('Fill error: ' + err.response?.data || err.message);
        }
    };

    return (
        <div>
            <h2>Fill & Flatten a Field</h2>
            <input
                type="text"
                placeholder="Field name"
                value={name}
                onChange={e => setName(e.target.value)}
            />
            <input
                type="text"
                placeholder="Value"
                value={value}
                onChange={e => setValue(e.target.value)}
            />
            <button onClick={handleSubmit}>Submit</button>
            {submitted && (
                <div style={{ marginTop: 10 }}>
                    ✅ PDF flattened and opened in new tab.
                </div>
            )}
        </div>
    );
};

export default FillForm;
