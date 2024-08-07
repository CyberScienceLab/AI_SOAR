<h1 align="center"> AI-SOAR </h1>

![LLM Gateway Routing For Shuffler UI Apps PDF Title Page](assets\Cyber_Science_Lab_Banner.png)

## Applications

Explore the applications and tutorials for AI-SOAR in our [document](assets\Shuffler_LLM_Gateway_Routing.pdf).

For a more interactive learning experience, check out our [YouTube Playlist](https://www.youtube.com/playlist?list=PLl2a3mDFCjeObCiZ9p8vASha5tM20fORC). This playlist features the following videos:

1. **[Creating Gemini App in Shuffler](https://youtu.be/wJheKNjDPT4?si=jh42e3focXs22tmN)**
    ![Creating Gemini App in Shuffler Thumbnail](assets\Shuffler_Gemini_App_Thumbnail.png)

2. **[Simple Shuffler Gemini Workflow](https://youtu.be/uiB_45pE2co?si=aXFUWg4YNEzAGxdj)**
    ![Simple Shuffler Gemini Workflow Thumbnail](assets\Shuffler_Simple_Gemini_Workflow_Thumbnail.png)

3. **[Shuffler Phishing Workflow](https://youtu.be/7pB_iw3mpPE?si=qhMmmQeo2j-ARyRw)**
    ![Shuffler Phishing Workflow Thumbnail](assets\Shuffler_Phishing_Workflow_Thumbnail.png)

4. **[Shuffler RAG CVE Workflow](https://youtu.be/Zdont8taRfg?si=dzmvO6UOyVLDsPW5)**
    ![Shuffler RAG CVE Workflow Thumbnail](assets\Shuffler_RAG_CVE_Workflow_Thumbnail.png)

## Run AI_SOAR for first time
1) In terminal: cd Shuffler
2) Create .env file and copy contents from .env.example. Must assign values to following fields:
    -  GATEWAY_URL
3) Build backend image
    - In terminal: docker build -t csl-backend:latest /backend
4) Build frontend image
    - In terminal: docker build -t csl-frontend:latest /frontend
3) In terminal: docker compose up -d
