http://localhost:8080/login.html

Git으로 커밋(commit) 해줘..env는 커밋하지말고,메시지는 네가 작성해줘.

방금 Gemini가 만든 코드를 커밋했으니, 
이어서 git status, git diff 같은 명령어로 변경 사항을 확인하고,
 read_file로 코드를 읽어 컨텍스트를 파악한 후
리뷰나 보완할 문제점을 체크해서 나에게 알려줘.


터미널에 서버작동여부를 확인하는 방법
ssh root@134.209.214.151
cd /root/politisian
docker-compose logs -f app

./politisian_server  로컬서버시작

mcp desktop command---mcp들을 연결해주는것 같음.그리고 깃허브의 프로그램 주소를 알려주면 자동으로 설치,연결해주는 역할

http://politisian.org/login.html


  
   DigitalOcean Droplet (서버)에 SSH로 접속한 후, 다음 명령어를 터미널에 입력하여 politisian 관련
  프로세스가 실행되고 있는지 확인할 수 있습니다.

   1 ps aux | grep politisian

  명령어 설명:

   * ps aux: 현재 시스템에서 실행 중인 모든 프로세스를 자세히 보여줍니다.
   * | (파이프): 왼쪽 명령어(ps aux)의 결과를 오른쪽 명령어(grep politisian)의 입력으로
     전달합니다.
   * grep politisian: 입력받은 내용에서 'politisian'이라는 단어가 포함된 라인만 필터링하여
     보여줍니다.



     claude code 에서 일일 사용량을 확인하는 터미널 명령어 
     일별사용량    npx ccusage


    서버 업데이트 방법:
      
  ssh root@134.209.214.151
  cd /root/politisian
  git pull origin main
  docker-compose down
  docker-compose up -d
  docker-compose logs -f app 


  http://politisian.org/login.html