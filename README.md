# 📘 TogetherEdu – Manual
🌐 About  
TogetherEdu - a collaborative learning web platform designed for schoolchildren, students, and users who need a space for online interaction.  
🌍 Official website:  
https://tgreducation.duckdns.org  
The platform allows users to:  
•	👥 Create collaboration rooms  
•	💬 Use real-time chat  
•	🖊 Work on a shared online whiteboard  
•	🤖 Communicate with AI right inside the room  
TogetherEdu creates a convenient environment for group exam preparation, collaborative problem solving, and discussion of course material.  

🎯 Target audience  
•	Schoolchildren  
•	Universities  
•	Online tutors  
•	Users preparing for exams  
•	People working on collaborative projects  

🚀 Core Features  
Study Rooms  
•	Create private study rooms  
•	Limit the maximum number of participants  
•	Real-time participant tracking  
•	Live room status updates  
Real-Time Chat  
•	Instant message exchange  
•	WebSocket-based communication  
•	Live online user updates  
•	No page refresh required  
Collaborative Whiteboard  
•	Real-time drawing  
•	Shared canvas synchronization  
•	Multi-user interaction  
AI Integration  
•	Send questions directly to the AI assistant  
  
•	AI responses appear in the shared chat  
•	Backend API integration  
# Local Installation Guide  
Clone the Repository  
git clone https://github.com/desethh/FinalProject.git  
cd FinalProject  
Configure Environment Variables  
Create .env file:  
DB_HOST=localhost  
DB_PORT=5432  
DB_NAME=postgres  
DB_USER=postgres  
DB_PASSWORD=your_password  
API_KEY=your_ai_key  
Run Using Docker  
docker compose up -d --build  
After startup, open:  
http://localhost  

