import { useState } from 'react';
import Upload from './components/Upload';

function App() {
    const [fileName, setFileName] = useState<string | null>(null);

    const handleClear = () => setFileName(null);

    return (
        <div style={{ padding: 20 }}>
            <h1>Simple PDF Uploader</h1>

            <Upload onUploaded={setFileName} />

            {fileName && (
                <>
                    <hr />
                    <button onClick={handleClear}>Clear File</button>

                    <a
                        href={`http://localhost:8080/download/${fileName}`}
                        target="_blank"
                        rel="noreferrer"
                        style={{ marginLeft: 10 }}
                    >
                        Download {fileName}
                    </a>
                </>
            )}
        </div>
    );
}

export default App;
